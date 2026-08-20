package analysis

import (
	"maps"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func toAnalysisConfig(cfg *config.GovernanceConfig) *core.AnalysisConfig {
	if cfg == nil {
		return &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{}}
	}
	analyzers := make(map[string]core.AnalyzerConfig, len(cfg.Analyzers)+1)
	for id, ac := range cfg.Analyzers {
		analyzers[id] = core.AnalyzerConfig{
			Enabled: ac.Enabled,
			Params:  ac.Params,
		}
	}
	withEmojiParams(analyzers, cfg.Emoji)
	withMovePurityParams(analyzers, cfg.MovePurity)
	return &core.AnalysisConfig{
		Analyzers:        analyzers,
		IncludeGenerated: cfg.IncludeGenerated,
	}
}

func withEmojiParams(analyzers map[string]core.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	const commitAnalyzerID = "commit-lint"
	setParam(analyzers, commitAnalyzerID, "emoji_hard_window", emojiCfg.WindowSize)
	setParam(analyzers, commitAnalyzerID, "emoji_soft_window", emojiCfg.SoftWindowSize)
	setParam(analyzers, commitAnalyzerID, "emoji_suggestions", emojiCfg.SuggestionCount)
}

func withMovePurityParams(analyzers map[string]core.AnalyzerConfig, cfg config.MovePurityConfig) {
	const movePurityAnalyzerID = "commit-move-purity"
	setParam(analyzers, movePurityAnalyzerID, "min_similarity", cfg.MinSimilarity)
	setParam(analyzers, movePurityAnalyzerID, "severity", cfg.Severity)
}

func setParam(analyzers map[string]core.AnalyzerConfig, id, key string, value any) {
	entry, found := analyzers[id]
	if !found {
		entry = core.AnalyzerConfig{Enabled: true}
	}
	params := make(map[string]any, len(entry.Params)+1)
	maps.Copy(params, entry.Params)
	params[key] = value
	entry.Params = params
	analyzers[id] = entry
}
