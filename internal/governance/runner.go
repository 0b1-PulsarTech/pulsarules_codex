package governance

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *pass.AnalysisConfig {
	return &pass.AnalysisConfig{
		Analyzers:        cfg.Analyzers,
		IncludeGenerated: cfg.IncludeGenerated,
		TestGlobs:        cfg.TestGlobs,
		Overrides:        slices.Concat(cfg.SeverityRules, cfg.Overrides),
		Delegated:        enabledDelegated(cfg.Delegated, cfg.Analyzers),
	}
}

func enabledDelegated(
	entries []pass.DelegatedCommand, analyzers map[string]pass.AnalyzerConfig,
) []pass.DelegatedCommand {
	// Dropped here: the analyzers under internal/analyzer/ may not read Analyzers directly.
	return slices.DeleteFunc(slices.Clone(entries), func(entry pass.DelegatedCommand) bool {
		listed, found := analyzers[entry.ID]
		return found && !listed.Enabled
	})
}
