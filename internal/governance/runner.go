package governance

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *pass.AnalysisConfig {
	analyzers := make(map[string]pass.AnalyzerConfig, len(cfg.Analyzers))
	for id, ac := range cfg.Analyzers {
		analyzers[id] = pass.AnalyzerConfig{
			Enabled: ac.Enabled,
			Params:  ac.Params,
		}
	}
	return &pass.AnalysisConfig{
		Analyzers:        analyzers,
		IncludeGenerated: cfg.IncludeGenerated,
		TestGlobs:        cfg.TestGlobs,
		Overrides:        slices.Concat(cfg.SeverityRules, cfg.Overrides),
		Delegated:        enabledDelegated(cfg.Delegated, cfg.Analyzers),
	}
}

func enabledDelegated(
	entries []pass.DelegatedCommand, analyzers map[string]config.AnalyzerConfig,
) []pass.DelegatedCommand {
	// Dropped here: the analyzers under internal/analyzer/ may not read Analyzers directly.
	return slices.DeleteFunc(slices.Clone(entries), func(entry pass.DelegatedCommand) bool {
		listed, found := analyzers[entry.ID]
		return found && !listed.Enabled
	})
}
