package config

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type GovernanceConfig struct {
	Preset           string
	Analyzers        map[string]AnalyzerConfig
	IncludeGenerated bool
	SkipDirs         []string
	SeverityRules    []pass.Override
	TestGlobs        []string
	Overrides        []pass.Override
	Delegated        []pass.DelegatedCommand
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

func (govCfg *GovernanceConfig) SetSeverity(analyzerID, severity string) {
	// Rules are applied in order and the last match wins, so a later source beats an earlier one.
	govCfg.SeverityRules = append(govCfg.SeverityRules, pass.NewOverride(
		nil, map[string]pass.OverrideConfig{analyzerID: {Severity: severity}},
	))
}
