package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runCommitLint(inj remy.Injector, opts *version.Options) error {
	msg := opts.CommitMsg
	if msg == "" && opts.CommitFile != "" {
		// CommitFile is a CLI flag value the operator supplies intentionally.
		msgBytes, err := os.ReadFile(opts.CommitFile)
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
	repo, err := optionalRepository(inj)
	if err != nil {
		return err
	}
	var root string
	if repo != nil {
		root = repo.Root()
	}

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	catalog, err := remy.Get[*emoji.Catalog](inj)
	if err != nil {
		return fmt.Errorf("get emoji catalog: %w", err)
	}
	cfg, configFindings := loader.ForRoot(root)
	sess := governance.NewSession(repo, msg, idx, catalog, cfg)
	// An explicit empty status keeps this a pure message linter. A nil
	// status would have discovery read the worktree, and content findings from
	// unrelated dirty files would fail a commit whose message is fine.
	result := sess.Analyze(governance.ScopeCommit, &vcs.Status{}, governance.FileSetChanged)
	result.Findings = slices.Concat(configFindings, result.Findings)

	if len(result.Findings) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "commit message is valid")
		return nil
	}

	_, _ = fmt.Fprint(os.Stderr, result.Text(findings.CLI(), ""))

	if result.HasErrors() {
		return ErrFindings
	}
	return nil
}

//nolint:ireturn // returns the consumer port, nil when there is no repository
func optionalRepository(inj remy.Injector) (governance.Repository, error) {
	opened, err := remy.Get[*vcs.Repository](inj)
	if err != nil && !errors.Is(err, vcs.ErrNoRepository) {
		return nil, fmt.Errorf("open repository: %w", err)
	}
	// The one place an absent repository becomes a true nil interface: a nil
	// *vcs.Repository stored in the port would not compare equal to nil.
	if opened == nil {
		return nil, nil //nolint:nilnil // an absent repository is a valid, explicit nil
	}
	return opened, nil
}
