package governance

import (
	"maps"

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
	return &pass.AnalysisConfig{Analyzers: analyzers}
}

func withEmojiParams(analyzers map[string]pass.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	const commitAnalyzerID = "commit-lint"

	entry, found := analyzers[commitAnalyzerID]
	if !found {
		entry = pass.AnalyzerConfig{Enabled: true}
	}
	params := make(map[string]any, len(entry.Params)+3)
	maps.Copy(params, entry.Params)
	params["emoji_hard_window"] = emojiCfg.WindowSize
	params["emoji_soft_window"] = emojiCfg.SoftWindowSize
	params["emoji_suggestions"] = emojiCfg.SuggestionCount

	entry.Params = params
	analyzers[commitAnalyzerID] = entry
}
