package config

import "maps"

const (
	PresetRecommended = "recommended"
	PresetStrict      = "strict"
	PresetMinimal     = "minimal"
)

func presetAnalyzers() map[string]map[string]AnalyzerConfig {
	// Only overrides relative to the default are listed; absent analyzers stay enabled.
	return map[string]map[string]AnalyzerConfig{
		PresetMinimal: minimalAnalyzers(),
		PresetStrict:  strictAnalyzers(),
	}
}

func minimalAnalyzers() map[string]AnalyzerConfig {
	return map[string]AnalyzerConfig{
		// AST analyzers are disabled in minimal mode.
		"control-flow": {Enabled: false},
	}
}

func strictAnalyzers() map[string]AnalyzerConfig {
	const (
		strictMaxFileLines = 180
	)
	return map[string]AnalyzerConfig{
		// Same number as the analyzer's default; stated here anyway since this
		// is the policy layer a caller reads to learn the project's limit.
		"file-size": {Enabled: true, Params: map[string]any{"max_lines": strictMaxFileLines}},
	}
}

func (govCfg *GovernanceConfig) ApplyPreset() {
	overrides, ok := presetAnalyzers()[govCfg.Preset]
	if !ok {
		return
	}
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = make(map[string]AnalyzerConfig)
	}
	maps.Copy(govCfg.Analyzers, overrides)
}
