package config

import "maps"

const (
	PresetRecommended = "recommended"
	PresetStrict      = "strict"
	PresetMinimal     = "minimal"
)

func presetTable() map[string]map[string]AnalyzerConfig {
	// Only overrides relative to the default are listed; absent analyzers stay enabled.
	return map[string]map[string]AnalyzerConfig{
		PresetMinimal: minimalAnalyzers(),
		PresetStrict:  strictAnalyzers(),
	}
}

func minimalAnalyzers() map[string]AnalyzerConfig {
	return map[string]AnalyzerConfig{}
}

func strictAnalyzers() map[string]AnalyzerConfig {
	return map[string]AnalyzerConfig{}
}

func (govCfg *GovernanceConfig) ApplyPreset() {
	overrides, ok := presetTable()[govCfg.Preset]
	if !ok {
		return
	}
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = make(map[string]AnalyzerConfig)
	}
	maps.Copy(govCfg.Analyzers, overrides)
}
