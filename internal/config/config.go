package config

// GovernanceConfig is the top-level configuration for the governance framework.
// It controls which analyzers are enabled, their thresholds, and the emoji
// rules for commit validation. Defaults are embedded in the binary; presets
// modify them in-memory at runtime.
type GovernanceConfig struct {
	// Preset selects a named configuration preset (strict, recommended,
	// minimal). Empty defaults to recommended.
	Preset string
	// Analyzers maps analyzer ID to its runtime config.
	Analyzers map[string]AnalyzerConfig
	// Emoji holds the commit emoji configuration.
	Emoji EmojiConfig
	// MovePurity holds the commit-move-purity analyzer's configuration.
	MovePurity MovePurityConfig
	// IncludeGenerated keeps findings that fall in files carrying the Go
	// generated-code marker. Off by default: nobody can act on them, because
	// the next codegen run overwrites the fix.
	IncludeGenerated bool
}

// AnalyzerConfig holds the per-analyzer enable/disable state and parameters.
// It is field-for-field identical to core.AnalyzerConfig today, deliberately:
// this is the shape a preset writes, core's is what the stage runner reads,
// and keeping them apart lets internal/analyzer/core stay stdlib-only (see
// analysis.toAnalysisConfig, which projects emoji/move-purity into Params).
type AnalyzerConfig struct {
	// Enabled controls whether the analyzer runs. Defaults to true when
	// absent from the map.
	Enabled bool
	// Params is an arbitrary key-value map for analyzer-specific settings
	// (e.g. max_file_lines=180, max_complexity=15).
	Params map[string]any
}

// EmojiConfig holds the commit emoji validation configuration.
type EmojiConfig struct {
	// WindowSize is the number of preceding commits an emoji may not repeat
	// within. A violation blocks the commit. Default: 5.
	WindowSize int
	// SoftWindowSize is the wider span that only earns advice. Default: 20.
	SoftWindowSize int
	// SuggestionCount is how many alternatives a finding offers. Default: 7.
	SuggestionCount int
}

// MovePurityConfig holds the commit-move-purity analyzer's configuration:
// how similar a staged rename must be to count as pure, and how hard a
// violation lands.
type MovePurityConfig struct {
	// MinSimilarity is the minimum git rename-similarity score (0-100) a
	// staged rename must meet to count as pure. Default: 90.
	MinSimilarity int
	// Severity controls how a finding affects the hook's exit behavior:
	// "warning" (default) never blocks; "error" fails a non-pure or mixed
	// commit at the pre-commit gate.
	Severity string
}

const (
	defaultEmojiWindowSize      = 5
	defaultEmojiSoftWindowSize  = 20
	defaultEmojiSuggestionCount = 7
	defaultMoveMinSimilarity    = 90
)

// Defaults returns the default governance configuration embedded in the
// binary. All analyzers are enabled by default; an emoji repeating within five
// commits blocks, and one repeating within twenty only draws advice.
func Defaults() *GovernanceConfig {
	return &GovernanceConfig{
		Preset:    PresetRecommended,
		Analyzers: map[string]AnalyzerConfig{},
		Emoji: EmojiConfig{
			WindowSize:      defaultEmojiWindowSize,
			SoftWindowSize:  defaultEmojiSoftWindowSize,
			SuggestionCount: defaultEmojiSuggestionCount,
		},
		MovePurity: MovePurityConfig{
			MinSimilarity: defaultMoveMinSimilarity,
			Severity:      "warning",
		},
	}
}

// Param returns a parameter value for an analyzer, or the default if unset.
//
// why: production reads params via core.AnalysisConfig/ParamSet instead;
// this survives only because governance_test.go still asserts through it -
// IsEnabled, lacking that dependency, was deleted alongside it.
func (c *GovernanceConfig) Param(analyzerID, key string, def any) any {
	cfg, ok := c.Analyzers[analyzerID]
	if !ok {
		return def
	}
	val, ok := cfg.Params[key]
	if !ok {
		return def
	}
	return val
}

// SetParam sets one analyzer's runtime parameter, creating the analyzer's entry
// when it has none, so a caller never has to know how the nested maps are built.
func (c *GovernanceConfig) SetParam(analyzerID, key string, value any) {
	if c.Analyzers == nil {
		c.Analyzers = map[string]AnalyzerConfig{}
	}
	cfg, ok := c.Analyzers[analyzerID]
	if !ok {
		cfg = AnalyzerConfig{Enabled: true}
	}
	if cfg.Params == nil {
		cfg.Params = map[string]any{}
	}
	cfg.Params[key] = value
	c.Analyzers[analyzerID] = cfg
}
