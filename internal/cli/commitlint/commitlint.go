package commitlint

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Command struct {
	command.Project
	msg  string
	file string
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.StringVar(&cmd.msg, "msg", "", "commit message `MSG` to validate")
	fs.StringVar(&cmd.file, "file", "", "path to a COMMIT_EDITMSG `FILE`")
	cmd.BindProject(fs, "project `DIR` for git history lookup")
}

func (cmd *Command) Run(
	openRepository func() (*vcs.Repository, error),
	index *knowledge.Index,
	loadCatalog func() (*emoji.Catalog, error),
	newLayered func() *loader.Layered,
) error {
	msg := cmd.msg
	if msg == "" && cmd.file != "" {
		// The file is a CLI flag value the operator supplies intentionally.
		msgBytes, err := os.ReadFile(cmd.file)
		if err != nil {
			return fmt.Errorf("read commit file: %w", err)
		}
		msg = string(msgBytes)
	}
	if msg == "" {
		return errors.New("provide --msg or --file")
	}

	// Without a repository there is no git history, so the emoji-repetition
	// window silently passes; any other resolution failure (e.g. a DI
	// wiring bug) must still surface as an error.
	repo, err := command.OptionalRepository(openRepository)
	if err != nil {
		return err
	}
	var root string
	if repo != nil {
		root = repo.Root()
	}

	catalog, err := loadCatalog()
	if err != nil {
		return fmt.Errorf("get emoji catalog: %w", err)
	}
	cfg, configFindings := newLayered().ForRoot(root)
	sess := governance.NewSession(repo, index, catalog, cfg)
	// An explicit empty status keeps this a pure message linter. A nil
	// status would have discovery read the worktree, and content findings from
	// unrelated dirty files would fail a commit whose message is fine.
	result := sess.Analyze(governance.Request{
		Scope:     governance.ScopeCommit,
		Status:    &vcs.Status{},
		Files:     governance.FileSetChanged,
		CommitMsg: msg,
	})
	result.Findings = slices.Concat(configFindings, result.Findings)

	if len(result.Findings) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "commit message is valid")
		return nil
	}

	_, _ = fmt.Fprint(os.Stderr, result.Text(findings.CLI(), ""))

	if result.HasErrors() {
		return command.ErrFindings
	}
	return nil
}
