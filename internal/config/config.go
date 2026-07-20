package config

type GovernanceConfig struct {
	Preset    string
	Analyzers map[string]AnalyzerConfig
	Emoji     EmojiConfig
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

func Defaults() *GovernanceConfig {
	const (
		defaultEmojiWindowSize      = 5
		defaultEmojiSoftWindowSize  = 20
		defaultEmojiSuggestionCount = 7
	)

	return &GovernanceConfig{
		Preset:    PresetRecommended,
		Analyzers: map[string]AnalyzerConfig{},
		Emoji: EmojiConfig{
			WindowSize:      defaultEmojiWindowSize,
			SoftWindowSize:  defaultEmojiSoftWindowSize,
			SuggestionCount: defaultEmojiSuggestionCount,
		},
	}
}
