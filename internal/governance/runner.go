package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *pass.AnalysisConfig {
	if cfg == nil {
		return &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{}}
	}
	analyzers := make(map[string]pass.AnalyzerConfig, len(cfg.Analyzers))
	for id, ac := range cfg.Analyzers {
		analyzers[id] = pass.AnalyzerConfig{
			Enabled: ac.Enabled,
			Params:  ac.Params,
		}
	}
	return &pass.AnalysisConfig{Analyzers: analyzers}
}
