package analysis

import (
	"maps"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

// toAnalysisConfig converts a GovernanceConfig into the AnalysisConfig used by
// the stage runner's enabled() check.
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
	return &core.AnalysisConfig{Analyzers: analyzers}
}

// withEmojiParams projects the emoji settings onto the commit analyzer's
// params, which is how analyzers read runtime configuration.
func withEmojiParams(analyzers map[string]core.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	const commitAnalyzerID = "commit-lint"

	entry, found := analyzers[commitAnalyzerID]
	if !found {
		entry = core.AnalyzerConfig{Enabled: true}
	}
	params := make(map[string]any, len(entry.Params)+3)
	maps.Copy(params, entry.Params)
	params["emoji_hard_window"] = emojiCfg.WindowSize
	params["emoji_soft_window"] = emojiCfg.SoftWindowSize
	params["emoji_suggestions"] = emojiCfg.SuggestionCount

	entry.Params = params
	analyzers[commitAnalyzerID] = entry
}

// withMovePurityParams projects the move-purity settings onto the analyzer's
// own params, the same way withEmojiParams does for commit-lint.
func withMovePurityParams(analyzers map[string]core.AnalyzerConfig, cfg config.MovePurityConfig) {
	const movePurityAnalyzerID = "commit-move-purity"

	entry, found := analyzers[movePurityAnalyzerID]
	if !found {
		entry = core.AnalyzerConfig{Enabled: true}
	}
	params := make(map[string]any, len(entry.Params)+2)
	maps.Copy(params, entry.Params)
	params["min_similarity"] = cfg.MinSimilarity
	params["severity"] = cfg.Severity

	entry.Params = params
	analyzers[movePurityAnalyzerID] = entry
}
