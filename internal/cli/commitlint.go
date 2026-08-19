package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
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
	repo, err := remy.Get[vcs.Repository](inj)
	if err != nil && !errors.Is(err, vcs.ErrNoRepository) {
		return fmt.Errorf("open repository: %w", err)
	}

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	sess := governance.NewSession(repo, msg, idx, nil)
	// An explicit empty status keeps this a pure message linter. A nil
	// status would have discovery read the worktree, and content findings from
	// unrelated dirty files would fail a commit whose message is fine.
	findingsItem := sess.Analyze(
		governance.ScopeCommit,
		&vcs.Status{},
		governance.FileSetChanged,
	).Findings

	if len(findingsItem) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "commit message is valid")
		return nil
	}

	_, _ = fmt.Fprint(os.Stderr, findings.FormatFindings(findingsItem, findings.StyleCLI, ""))

	if errCount, _, _ := governance.CountBySeverity(findingsItem); errCount > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}
