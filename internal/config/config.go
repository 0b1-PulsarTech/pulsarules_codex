package config

import (
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type GovernanceConfig struct {
	pass.AnalysisConfig
	Preset        string
	SkipDirs      []string
	SeverityRules []pass.Override
}

func Defaults() *GovernanceConfig {
	return &GovernanceConfig{
		Preset:    PresetRecommended,
		Analyzers: map[string]pass.AnalyzerConfig{},
	}
}

func (govCfg *GovernanceConfig) Analysis() *pass.AnalysisConfig {
	built := govCfg.AnalysisConfig
	// A fresh map, so adding or replacing an entry on either config never shows on the other.
	built.Analyzers = maps.Clone(govCfg.Analyzers)
	// Severity rules come first, so a per-path override beats a project or flag severity.
	built.Overrides = slices.Concat(govCfg.SeverityRules, govCfg.Overrides)
	// Dropped here: the analyzers under internal/analyzer/ may not read Analyzers directly.
	built.Delegated = slices.DeleteFunc(
		slices.Clone(govCfg.Delegated),
		func(entry pass.DelegatedCommand) bool { return !govCfg.Enabled(entry.ID) },
	)
	return &built
}

func (govCfg *GovernanceConfig) SetParam(analyzerID, key string, value any) {
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = map[string]pass.AnalyzerConfig{}
	}
	cfg, ok := govCfg.Analyzers[analyzerID]
	if !ok {
		cfg = pass.AnalyzerConfig{Enabled: true}
	}
	if cfg.Params == nil {
		cfg.Params = map[string]any{}
	}
	cfg.Params[key] = value
	govCfg.Analyzers[analyzerID] = cfg
}

func (govCfg *GovernanceConfig) SetSeverity(analyzerID, severity string) {
	// Rules are applied in order and the last match wins, so a later source beats an earlier one.
	govCfg.SeverityRules = append(govCfg.SeverityRules, pass.NewOverride(
		nil, map[string]pass.OverrideConfig{analyzerID: {Severity: severity}},
	))
}

const AnalyzerID = "config"

func Finding(file, message, suggestion string) finding.Finding {
	return finding.NewReporter(AnalyzerID, finding.SeverityError, finding.CatProject).At(
		file, 0, message, suggestion,
	)
}
