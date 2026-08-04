package cli

import (
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// runCommitLint validates a commit message via the session's commit-only
// analysis path. The message comes from --msg (a direct string) or --file (a
// path to COMMIT_EDITMSG). It prints findings to stderr and returns an
// *ExitError{Code: 1} if any error-severity finding is produced; main is the
// only caller that turns that into os.Exit.
func runCommitLint(inj remy.Injector, opts *cliopts.Options) error {
	msg := opts.CommitMsg
	if msg == "" && opts.CommitFile != "" {
		//nolint:gosec // CLI flag value, user intentionally specifying path
		data, err := os.ReadFile(opts.CommitFile)
		if err != nil {
			return fmt.Errorf("read commit file: %w", err)
		}
		msg = string(data)
	}
	if msg == "" {
		return fmt.Errorf("provide --msg or --file")
	}

	// Without a project dir, or when it is not a git repository, there is no
	// git history, and the emoji repetition window silently passes
	// everything rather than failing the commit over a missing repo.
	repo, _ := remy.Get[vcs.Repository](inj)

	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	sess := analysis.NewSession(repo, msg, idx, nil)
	findings := sess.Analyze(analysis.ScopeCommit, nil, analysis.FileSetChanged).Findings

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
