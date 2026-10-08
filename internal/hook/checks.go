package hook

import (
	"fmt"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis/projectcfg"
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
	cfg, configFindings := projectcfg.ForRepo(repo)
	sess := analysis.NewSession(repo, "", index, cfg)
	result := sess.Analyze(scope, &status, analysis.FileSetChanged)
	findings := slices.Concat(configFindings, result.Findings)

	block := analysis.FormatFindings(findings, analysis.StyleHook, "Governance checks:")
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
	return "\n" + block, len(findings)
}
