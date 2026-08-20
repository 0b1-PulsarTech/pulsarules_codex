package governance

import (
	"maps"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *pass.AnalysisConfig {
	if cfg == nil {
		return &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{}}
	}
	analyzers := make(map[string]pass.AnalyzerConfig, len(cfg.Analyzers)+1)
	for id, ac := range cfg.Analyzers {
		analyzers[id] = pass.AnalyzerConfig{
			Enabled: ac.Enabled,
			Params:  ac.Params,
		}
	}
	withEmojiParams(analyzers, cfg.Emoji)
	withMovePurityParams(analyzers, cfg.MovePurity)
	return &pass.AnalysisConfig{
		Analyzers:        analyzers,
		IncludeGenerated: cfg.IncludeGenerated,
	}
}

func withEmojiParams(analyzers map[string]pass.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	setParam(analyzers, message.ID, message.ParamEmojiHardWindow, emojiCfg.WindowSize)
	setParam(
		analyzers,
		message.ID,
		message.ParamEmojiSoftWindow,
		emojiCfg.SoftWindowSize,
	)
	setParam(
		analyzers, message.ID, message.ParamEmojiSuggestions, emojiCfg.SuggestionCount,
	)
}

func withMovePurityParams(analyzers map[string]pass.AnalyzerConfig, cfg config.MovePurityConfig) {
	setParam(
		analyzers,
		movepurity.ID,
		movepurity.ParamMinSimilarity,
		cfg.MinSimilarity,
	)
	setParam(analyzers, movepurity.ID, pass.ParamSeverity, cfg.Severity)
}

func setParam(analyzers map[string]pass.AnalyzerConfig, id, key string, value any) {
	entry, found := analyzers[id]
	if !found {
		entry = pass.AnalyzerConfig{Enabled: true}
	}
	params := make(map[string]any, len(entry.Params)+1)
	maps.Copy(params, entry.Params)
	params[key] = value
	entry.Params = params
	analyzers[id] = entry
}
