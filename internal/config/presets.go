package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	PresetRecommended = "recommended"
	PresetStrict      = "strict"
	PresetMinimal     = "minimal"
)

type preset struct {
	name      string
	analyzers func() map[string]pass.AnalyzerConfig
}

func presetTable() []preset {
	// Only overrides relative to the default are listed; absent analyzers stay enabled.
	return []preset{
		{name: PresetRecommended},
		{name: PresetStrict, analyzers: strictAnalyzers},
		{name: PresetMinimal, analyzers: minimalAnalyzers},
	}
}

func minimalAnalyzers() map[string]pass.AnalyzerConfig {
	return map[string]pass.AnalyzerConfig{
		// AST and arch analyzers are disabled in minimal mode.
		"control-flow":       {Enabled: false},
		"shadowing":          {Enabled: false},
		"complexity":         {Enabled: false},
		"magic-number":       {Enabled: false},
		"named-returns":      {Enabled: false},
		"context-discipline": {Enabled: false},
		// pprof-labels stays enabled even in minimal: it catches PII leaking
		// into panic tracebacks, so the lightest preset must not disable it.
		"arch-boundary": {Enabled: false},
		"import-cycle":  {Enabled: false},
		golangcilint.ID: {Enabled: false},
		// First-adoption noise: both flag hundreds of tests and declarations
		// written before their rules.
		"test-comment":           {Enabled: false},
		"single-use-declaration": {Enabled: false},
		// First-adoption noise: these analyzers flag existing code written before
		// the rules they enforce.
		"doc-comment":          {Enabled: false},
		"body-header-comment":  {Enabled: false},
		"history-narration":    {Enabled: false},
		"variadic-index":       {Enabled: false},
		"too-many-results":     {Enabled: false},
		"ungrouped-interfaces": {Enabled: false},
		"package-level-var":    {Enabled: false},
		"package-file-count":   {Enabled: false},
		// Minimal keeps upper-case single letters on funcs, methods, types, consts,
		// vars and fields.
		"single-letter-name": {Enabled: true, Params: map[string]any{"exported_only": true}},
	}
}

func strictAnalyzers() map[string]pass.AnalyzerConfig {
	const (
		strictMaxFileLines = 180
		strictMaxFuncLines = 50
		strictMaxParams    = 4
		strictComplexity   = 10
	)
	return map[string]pass.AnalyzerConfig{
		// Same number as the analyzer's default; stated here anyway since this
		// is the policy layer a caller reads to learn the project's limit.
		filesize.ID: {Enabled: true, Params: map[string]any{
			filesize.ParamMaxLines: strictMaxFileLines,
		}},
		"complexity": {Enabled: true, Params: map[string]any{
			"max_complexity": strictComplexity,
			"max_func_lines": strictMaxFuncLines,
			"max_params":     strictMaxParams,
		}},
	}
}

func (govCfg *GovernanceConfig) ApplyPreset() {
	// Unknown presets are left alone; callers validate the name before this.
	for _, listed := range presetTable() {
		if listed.name != govCfg.Preset || listed.analyzers == nil {
			continue
		}
		if govCfg.Analyzers == nil {
			govCfg.Analyzers = make(map[string]pass.AnalyzerConfig)
		}
		maps.Copy(govCfg.Analyzers, listed.analyzers())
	}
}

func ValidPreset(name string) bool {
	return slices.Contains(Presets(), name)
}

func Presets() []string {
	table := presetTable()
	names := make([]string, len(table))
	for idx, listed := range table {
		names[idx] = listed.name
	}
	return names
}

func UnknownPreset(subject, name string) string {
	return fmt.Sprintf("%s %q (want %s)", subject, name, strings.Join(Presets(), "|"))
}
