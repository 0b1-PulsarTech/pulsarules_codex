package config

import (
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func (govCfg *GovernanceConfig) ApplyProject(file *ProjectFile) error {
	if file.IncludeGenerated != nil {
		govCfg.IncludeGenerated = *file.IncludeGenerated
	}
	govCfg.SkipDirs = append(govCfg.SkipDirs, file.SkipDirs...)
	govCfg.TestGlobs = slices.Clone(file.Tests)
	for id, entry := range file.Analyzers {
		govCfg.applyAnalyzer(id, entry)
	}
	for _, override := range file.Overrides {
		govCfg.Overrides = append(govCfg.Overrides, toCoreOverride(override))
	}
	for _, entry := range file.Delegated {
		govCfg.Delegated = append(govCfg.Delegated, core.DelegatedCommand{
			ID:     entry.ID,
			Dir:    entry.Dir,
			Format: entry.Format,
			Rule:   entry.Rule,
			Run:    slices.Clone(entry.Run),
		})
	}
	return govCfg.CheckOverrides(file)
}

func (govCfg *GovernanceConfig) CheckOverrides(file *ProjectFile) error {
	problems := &collector{path: file.Path}
	listed := newRoot().key("overrides")
	for idx, override := range file.Overrides {
		for _, id := range slices.Sorted(maps.Keys(override.Analyzers)) {
			rules := override.Analyzers[id]
			base, _, _ := strings.Cut(id, "/")
			current, found := govCfg.Analyzers[base]
			if rules.Enabled == nil || !*rules.Enabled || !found || current.Enabled {
				continue
			}
			field := listed.item(idx).key("analyzers").key(id).key("enabled")
			problems.add(0, field, "%s is disabled for the whole project; "+
				"an override can only narrow it", clipValue(base))
		}
	}
	return problems.err()
}

func (govCfg *GovernanceConfig) applyAnalyzer(id string, entry AnalyzerEntry) {
	if govCfg.Analyzers == nil {
		govCfg.Analyzers = map[string]AnalyzerConfig{}
	}
	current, found := govCfg.Analyzers[id]
	if !found {
		current = AnalyzerConfig{Enabled: true}
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
		if govCfg.Severities == nil {
			govCfg.Severities = map[string]string{}
		}
		govCfg.Severities[id] = entry.Severity
	}
}

func toCoreOverride(entry OverrideEntry) core.Override {
	analyzers := make(map[string]core.OverrideConfig, len(entry.Analyzers))
	for id, rules := range entry.Analyzers {
		analyzers[id] = core.OverrideConfig{Enabled: rules.Enabled, Severity: rules.Severity}
	}
	return core.Override{Files: slices.Clone(entry.Files), Analyzers: analyzers}
}

func (val *validator) checkOverrides(entries []OverrideEntry) {
	field := val.root.key("overrides")
	for idx, override := range entries {
		val.checkOverride(field.item(idx), override)
	}
}

func (val *validator) checkOverride(field *fieldPath, override OverrideEntry) {
	if len(override.Files) == 0 {
		val.fail(field.key("files"), "at least one glob is required")
	}
	for _, id := range slices.Sorted(maps.Keys(override.Analyzers)) {
		rules := override.Analyzers[id]
		idField := field.key("analyzers").key(id)
		val.checkOverrideAnalyzer(idField, id)
		val.checkSeverity(idField.key("severity"), rules.Severity)
		if rules.Params != nil {
			val.fail(idField.key("params"), "per-path params are not supported")
		}
	}
}

func (val *validator) checkGlobBudget(globs []string) bool {
	size, wildcards, tails := 0, 0, 0
	for _, glob := range globs {
		size += len(glob)
		wildcards += wildcardWork(glob)
		tails += starTail(glob)
	}
	at := val.root.key("overrides and tests")
	switch {
	case len(globs) > MaxTotalGlobs:
		val.fail(at, "more than %d globs in total", MaxTotalGlobs)
	case size > MaxTotalGlobBytes:
		val.fail(at, "globs exceed %d bytes in total", MaxTotalGlobBytes)
	case wildcards > MaxWildcardWork:
		val.fail(at, "wildcard classes and ? exceed %d in total", MaxWildcardWork)
	case tails > MaxStarTextBytes:
		val.fail(at, "text after the first * of each segment exceeds %d bytes in total",
			MaxStarTextBytes)
	default:
		return true
	}
	return false
}
