package hook

import (
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
	findings := sess.Analyze(scope, &status, analysis.FileSetChanged)

	block := analysis.FormatFindings(findings, analysis.StyleHook, "Governance checks:")
	if block == "" {
		return "", 0
	}
	return "\n" + block, len(findings)
}
