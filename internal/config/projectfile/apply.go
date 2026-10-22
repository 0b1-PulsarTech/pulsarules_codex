package projectfile

import (
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
)

func (file *File) ApplyTo(govCfg *config.GovernanceConfig) error {
	if file.IncludeGenerated != nil {
		govCfg.IncludeGenerated = *file.IncludeGenerated
	}
	govCfg.SkipDirs = append(govCfg.SkipDirs, file.SkipDirs...)
	govCfg.TestGlobs = slices.Clone(file.Tests)
	for id, entry := range file.Analyzers {
		applyAnalyzer(govCfg, id, entry)
	}
	for _, override := range file.Overrides {
		govCfg.Overrides = append(govCfg.Overrides, toCoreOverride(override))
	}
	for _, entry := range file.Delegated {
		// Same fields as pass.DelegatedCommand; the yaml tags live on the file side only.
		// The conversion copies reference fields shallowly: clone any new one below.
		command := pass.DelegatedCommand(entry)
		command.Run = slices.Clone(entry.Run)
		govCfg.Delegated = append(govCfg.Delegated, command)
	}
	return file.CheckOverrides(govCfg)
}

func (file *File) CheckOverrides(govCfg *config.GovernanceConfig) error {
	problems := strictyaml.NewProblems(file.Path)
	listed := strictyaml.Root().Key("overrides")
	for idx, override := range file.Overrides {
		for _, id := range slices.Sorted(maps.Keys(override.Analyzers)) {
			rules := override.Analyzers[id]
			base, _, _ := strings.Cut(id, "/")
			current, found := govCfg.Analyzers[base]
			if rules.Enabled == nil || !*rules.Enabled || !found || current.Enabled {
				continue
			}
			field := listed.Item(idx).Key("analyzers").Key(id).Key("enabled")
			problems.Report(field, "%s is disabled for the whole project; "+
				"an override can only narrow it", strictyaml.ClipValue(base))
		}
	}
	return problems.Err()
}

func applyAnalyzer(govCfg *config.GovernanceConfig, id string, entry AnalyzerEntry) {
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = map[string]pass.AnalyzerConfig{}
	}
	current, found := govCfg.Analyzers[id]
	if !found {
		current = pass.AnalyzerConfig{Enabled: true}
	}
	// Enabled is a pointer: absent keeps the current choice, false switches the analyzer off.
	if entry.Enabled != nil {
		current.Enabled = *entry.Enabled
	}
	if len(entry.Params) > 0 {
		merged := make(map[string]any, len(current.Params)+len(entry.Params))
		maps.Copy(merged, current.Params)
		maps.Copy(merged, entry.Params)
		current.Params = merged
	}
	govCfg.Analyzers[id] = current
	if entry.Severity != "" {
		govCfg.SetSeverity(id, entry.Severity)
	}
}

func toCoreOverride(entry OverrideEntry) pass.Override {
	analyzers := make(map[string]pass.OverrideConfig, len(entry.Analyzers))
	for id, rules := range entry.Analyzers {
		// Same fields as pass.OverrideConfig; a field added to one breaks this conversion.
		// The conversion copies reference fields shallowly: clone any new one below.
		override := pass.OverrideConfig(rules)
		override.Params = maps.Clone(rules.Params)
		analyzers[id] = override
	}
	return pass.NewOverride(entry.Files, analyzers)
}
