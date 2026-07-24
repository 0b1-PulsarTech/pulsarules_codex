package hook

import (
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
	findingsItem := sess.Analyze(scope, &status, governance.FileSetChanged)

	block := findings.FormatFindings(findingsItem, findings.StyleHook, "Governance checks:")
	if block == "" {
		return "", 0
	}
	return "\n" + block, len(findingsItem)
}
