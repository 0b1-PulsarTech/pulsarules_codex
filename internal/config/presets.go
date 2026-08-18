package config

import "maps"

// Preset names used to select a group of analyzer configurations.
const (
	PresetRecommended = "recommended"
	PresetStrict      = "strict"
	PresetMinimal     = "minimal"
)

// presetAnalyzers maps preset name → analyzer ID → AnalyzerConfig. Only the
// overrides relative to the default are listed; missing analyzers keep their
// default state (enabled with standard params).
var presetAnalyzers = map[string]map[string]AnalyzerConfig{
	PresetMinimal: {
		// AST and arch analyzers are disabled in minimal mode.
		"control-flow":  {Enabled: false},
		"shadowing":     {Enabled: false},
		"complexity":    {Enabled: false},
		"named-returns": {Enabled: false},
		"arch-boundary": {Enabled: false},
		"import-cycle":  {Enabled: false},
		"golangci-lint": {Enabled: false},
		"gopls":         {Enabled: false},
	},
	PresetStrict: {
		// Same number as the analyzer's own default today. It is stated here
		// anyway because this is the policy layer: a caller reading the config
		// must get the project's limit, not whatever fallback it happened to
		// pass.
		"file-size": {Enabled: true, Params: map[string]any{"max_lines": 180}},
		"complexity": {Enabled: true, Params: map[string]any{
			"max_complexity": 10,
			"max_func_lines": 50,
			"max_params":     4,
		}},
	},
}

// ApplyPreset modifies the config's Analyzers map with the overrides defined
// for the selected preset. Unknown presets are silently ignored (the caller
// should validate before calling). Missing analyzers keep their existing config.
func (c *GovernanceConfig) ApplyPreset() {
	overrides, ok := presetAnalyzers[c.Preset]
	if !ok {
		return
	}
	if c.Analyzers == nil {
		c.Analyzers = make(map[string]AnalyzerConfig)
	}
	maps.Copy(c.Analyzers, overrides)
}

// ValidPreset reports whether name is a known preset identifier.
func ValidPreset(name string) bool {
	if name == PresetRecommended {
		return true
	}
	_, ok := presetAnalyzers[name]
	return ok
}

func Presets() []string {
	return []string{PresetRecommended, PresetStrict, PresetMinimal}
}
