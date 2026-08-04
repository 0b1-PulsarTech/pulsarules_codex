package hook

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// RunGovernanceCheck runs the analyzer pipeline for scope against repo and
// returns a formatted findings block plus the number of findings behind it.
// status is the already-computed worktree status (avoids a redundant read). A
// commit message is never threaded through here: it is validated in exactly
// one place, the git commit-msg hook.
func RunGovernanceCheck(
	repo vcs.Repository,
	status vcs.Status,
	index *knowledge.Index,
	scope analysis.Scope,
) (string, int) {
	sess := analysis.NewSession(repo, "", index, nil)
	result := sess.Analyze(scope, &status, analysis.FileSetChanged)

	block := analysis.FormatFindings(result.Findings, analysis.StyleHook, "Governance checks:")
	if block == "" {
		return "", 0
	}
	if result.SuppressedGenerated > 0 {
		// The hook has no footer of its own, so the suppression is stated here
		// rather than dropped: a filtered block must say it was filtered.
		block += fmt.Sprintf(
			"  %d finding(s) suppressed in generated files\n",
			result.SuppressedGenerated,
		)
	}
	return "\n" + block, len(result.Findings)
}
