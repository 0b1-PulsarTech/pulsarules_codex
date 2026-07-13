package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *core.AnalysisConfig {
	if cfg == nil {
		return &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{}}
	}
	analyzers := make(map[string]core.AnalyzerConfig, len(cfg.Analyzers))
	for id, ac := range cfg.Analyzers {
		analyzers[id] = core.AnalyzerConfig{
			Enabled: ac.Enabled,
			Params:  ac.Params,
		}
	}
	return &core.AnalysisConfig{Analyzers: analyzers}
}
