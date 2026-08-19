package analysis

import (
	"maps"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
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
	return &core.AnalysisConfig{
		Analyzers:        analyzers,
		IncludeGenerated: cfg.IncludeGenerated,
	}
}

// withEmojiParams projects the emoji settings onto the commit analyzer's
// params, which is how analyzers read runtime configuration.
func withEmojiParams(analyzers map[string]core.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	setDefaultParam(analyzers, commit.AnalyzerID, commit.ParamEmojiHardWindow, emojiCfg.WindowSize)
	setDefaultParam(
		analyzers,
		commit.AnalyzerID,
		commit.ParamEmojiSoftWindow,
		emojiCfg.SoftWindowSize,
	)
	setDefaultParam(
		analyzers, commit.AnalyzerID, commit.ParamEmojiSuggestions, emojiCfg.SuggestionCount,
	)
}

// withMovePurityParams projects the move-purity settings onto the analyzer's
// own params, the same way withEmojiParams does for commit-lint.
func withMovePurityParams(analyzers map[string]core.AnalyzerConfig, cfg config.MovePurityConfig) {
	setDefaultParam(
		analyzers,
		movepurity.AnalyzerID,
		movepurity.ParamMinSimilarity,
		cfg.MinSimilarity,
	)
	setDefaultParam(analyzers, movepurity.AnalyzerID, core.ParamSeverity, cfg.Severity)
}

// setDefaultParam projects a typed config field onto an analyzer's param map
// UNLESS the caller already set that key explicitly (e.g. via
// GovernanceConfig.SetParam): an explicit param always wins over the value a
// config projection would otherwise supply, so SetParam is never silently
// clobbered by the config-to-param hop that runs after it.
func setDefaultParam(analyzers map[string]core.AnalyzerConfig, id, key string, value any) {
	entry, found := analyzers[id]
	if !found {
		entry = core.AnalyzerConfig{Enabled: true}
	}
	if _, explicit := entry.Params[key]; explicit {
		return
	}
	params := make(map[string]any, len(entry.Params)+1)
	maps.Copy(params, entry.Params)
	params[key] = value
	entry.Params = params
	analyzers[id] = entry
}
