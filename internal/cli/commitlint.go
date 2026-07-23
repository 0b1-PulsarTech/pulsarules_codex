package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runCommitLint(inj remy.Injector, opts *cliopts.Options) error {
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
	sess := analysis.NewSession(repo, msg, idx, nil)
	findings := sess.Analyze(analysis.ScopeCommit, nil, analysis.FileSetChanged)

	if len(findings) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "commit message is valid")
		return nil
	}

	_, _ = fmt.Fprint(os.Stderr, analysis.FormatFindings(findings, analysis.StyleCLI, ""))

	if errCount, _, _ := analysis.CountBySeverity(findings); errCount > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}
