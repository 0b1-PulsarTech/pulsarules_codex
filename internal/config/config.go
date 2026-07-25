package config

type GovernanceConfig struct {
	Preset     string
	Analyzers  map[string]AnalyzerConfig
	Emoji      EmojiConfig
	MovePurity MovePurityConfig
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

type EmojiConfig struct {
	WindowSize      int
	SoftWindowSize  int
	SuggestionCount int
}

type MovePurityConfig struct {
	MinSimilarity int
	Severity      string
}

func Defaults() *GovernanceConfig {
	const (
		defaultEmojiWindowSize      = 5
		defaultEmojiSoftWindowSize  = 20
		defaultEmojiSuggestionCount = 7
		defaultMoveMinSimilarity    = 90
	)

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

func (govCfg *GovernanceConfig) SetParam(analyzerID, key string, value any) {
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = map[string]AnalyzerConfig{}
	}
	cfg, ok := govCfg.Analyzers[analyzerID]
	if !ok {
		cfg = AnalyzerConfig{Enabled: true}
	}
	if cfg.Params == nil {
		cfg.Params = map[string]any{}
	}
	cfg.Params[key] = value
	govCfg.Analyzers[analyzerID] = cfg
}
