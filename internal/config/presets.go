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
	return map[string]AnalyzerConfig{
		// AST and arch analyzers are disabled in minimal mode.
		"control-flow":       {Enabled: false},
		"shadowing":          {Enabled: false},
		"complexity":         {Enabled: false},
		"named-returns":      {Enabled: false},
		"context-discipline": {Enabled: false},
		"arch-boundary":      {Enabled: false},
		"import-cycle":       {Enabled: false},
		"golangci-lint":      {Enabled: false},
	}
}

func strictAnalyzers() map[string]AnalyzerConfig {
	const (
		strictMaxFileLines = 180
		strictMaxFuncLines = 50
		strictMaxParams    = 4
		strictComplexity   = 10
	)
	return map[string]AnalyzerConfig{
		// Same number as the analyzer's default; stated here anyway since this
		// is the policy layer a caller reads to learn the project's limit.
		"file-size": {Enabled: true, Params: map[string]any{"max_lines": strictMaxFileLines}},
		"complexity": {Enabled: true, Params: map[string]any{
			"max_complexity": strictComplexity,
			"max_func_lines": strictMaxFuncLines,
			"max_params":     strictMaxParams,
		}},
	}
}

func (govCfg *GovernanceConfig) ApplyPreset() {
	overrides, ok := presetTable()[govCfg.Preset]
	// Unknown presets are left alone; callers validate the name before this.
	if !ok {
		return
	}
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = make(map[string]AnalyzerConfig)
	}
	maps.Copy(govCfg.Analyzers, overrides)
}

func ValidPreset(name string) bool {
	if name == PresetRecommended {
		return true
	}
	_, ok := presetTable()[name]
	return ok
}

func Presets() []string {
	return []string{PresetRecommended, PresetStrict, PresetMinimal}
}
