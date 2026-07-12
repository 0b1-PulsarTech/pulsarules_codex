package config

type GovernanceConfig struct {
	Preset    string
	Analyzers map[string]AnalyzerConfig
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

func Defaults() *GovernanceConfig {
	return &GovernanceConfig{
		Preset:    PresetRecommended,
		Analyzers: map[string]AnalyzerConfig{},
	}
}
