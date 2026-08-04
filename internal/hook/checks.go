package hook

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RunGovernanceCheck(
	repo vcs.Repository,
	status vcs.Status,
	index *knowledge.Index,
	scope governance.Scope,
) (string, int) {
	// The empty commit message is deliberate: only the git commit-msg hook validates one.
	sess := governance.NewSession(repo, "", index, nil)
	result := sess.Analyze(scope, &status, governance.FileSetChanged)

	block := findings.FormatFindings(result.Findings, findings.StyleHook, "Governance checks:")
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
