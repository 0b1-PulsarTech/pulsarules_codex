package hook

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RunGovernanceCheck(
	repo vcs.Repository,
	status vcs.Status,
	index *knowledge.Index,
	scope analysis.Scope,
) (string, int) {
	// The empty commit message is deliberate: only the git commit-msg hook validates one.
	sess := analysis.NewSession(repo, "", index, nil)
	findings := sess.Analyze(scope, &status, analysis.FileSetChanged)

	block := analysis.FormatFindings(findings, analysis.StyleHook, "Governance checks:")
	if block == "" {
		return "", 0
	}
	return "\n" + block, len(findings)
}
