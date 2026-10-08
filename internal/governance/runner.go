package governance

import (
	"maps"
	"slices"

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
		TestGlobs:        cfg.TestGlobs,
		Overrides:        slices.Concat(severityOverrides(cfg.Severities), cfg.Overrides),
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

func severityOverrides(severities map[string]string) []pass.Override {
	// One rule with no file scope reaches analyzers that never read a severity param.
	overrides := make([]pass.Override, 0, len(severities))
	for _, id := range slices.Sorted(maps.Keys(severities)) {
		overrides = append(overrides, pass.Override{
			Analyzers: map[string]pass.OverrideConfig{id: {Severity: severities[id]}},
		})
	}
	return overrides
}

func withEmojiParams(analyzers map[string]pass.AnalyzerConfig, emojiCfg config.EmojiConfig) {
	setDefaultParam(analyzers, message.ID, message.ParamEmojiHardWindow, emojiCfg.WindowSize)
	setDefaultParam(
		analyzers,
		message.ID,
		message.ParamEmojiSoftWindow,
		emojiCfg.SoftWindowSize,
	)
	setDefaultParam(
		analyzers, message.ID, message.ParamEmojiSuggestions, emojiCfg.SuggestionCount,
	)
}

func withMovePurityParams(analyzers map[string]pass.AnalyzerConfig, cfg config.MovePurityConfig) {
	setDefaultParam(
		analyzers,
		movepurity.ID,
		movepurity.ParamMinSimilarity,
		cfg.MinSimilarity,
	)
	setDefaultParam(analyzers, movepurity.ID, pass.ParamSeverity, cfg.Severity)
}

func setDefaultParam(analyzers map[string]pass.AnalyzerConfig, id, key string, value any) {
	entry, found := analyzers[id]
	if !found {
		entry = pass.AnalyzerConfig{Enabled: true}
	}
	// An explicit param (e.g. via SetParam) wins over the projection.
	if _, explicit := entry.Params[key]; explicit {
		return
	}
	params := make(map[string]any, len(entry.Params)+1)
	maps.Copy(params, entry.Params)
	params[key] = value
	entry.Params = params
	analyzers[id] = entry
}
