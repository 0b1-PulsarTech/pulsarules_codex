package projectfile

import (
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
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
	problems := &Problems{path: file.Path}
	listed := Root().Key("overrides")
	for idx, override := range file.Overrides {
		for _, id := range slices.Sorted(maps.Keys(override.Analyzers)) {
			rules := override.Analyzers[id]
			base, _, _ := strings.Cut(id, "/")
			current, found := govCfg.Analyzers[base]
			if rules.Enabled == nil || !*rules.Enabled || !found || current.Enabled {
				continue
			}
			field := listed.Item(idx).Key("analyzers").Key(id).Key("enabled")
			problems.add(0, field, "%s is disabled for the whole project; "+
				"an override can only narrow it", ClipValue(base))
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

func (val *validator) checkOverrides(entries []OverrideEntry) {
	field := val.root.Key("overrides")
	for idx, override := range entries {
		val.checkOverride(field.Item(idx), override)
	}
}

func (val *validator) checkOverride(field *Field, override OverrideEntry) {
	if len(override.Files) == 0 {
		val.fail(field.Key("files"), "at least one glob is required")
	}
	for _, id := range slices.Sorted(maps.Keys(override.Analyzers)) {
		rules := override.Analyzers[id]
		idField := field.Key("analyzers").Key(id)
		val.checkOverrideAnalyzer(idField, id)
		val.checkSeverity(idField.Key("severity"), rules.Severity)
		val.checkOverrideParams(idField.Key("params"), id, rules.Params)
	}
}

func (val *validator) checkOverrideParams(
	field *Field, id string, params map[string]any,
) {
	if params == nil {
		return
	}
	if id != filesize.ID {
		val.fail(field, "per-path params are only supported for %s, not %s",
			filesize.ID, ClipValue(id))
		return
	}
	val.checkAnalyzerParams(field, id, params)
	if _, set := params[pass.ParamExtensions]; set {
		val.fail(field.Key(pass.ParamExtensions), "%s is not supported per path",
			pass.ParamExtensions)
	}
}

func (val *validator) checkAnalyzerParams(
	field *Field, id string, params map[string]any,
) {
	val.checkParams(field, id, params)
	if id == filesize.ID {
		val.checkLimitParams(field, params)
	}
}

func (val *validator) checkLimitParams(field *Field, params map[string]any) {
	soft := val.limitParam(field, params, filesize.ParamMaxLines, 1)
	hard := val.limitParam(field, params, filesize.ParamErrorLines, 0)
	if soft > 0 && hard > 0 && hard <= soft {
		val.fail(field.Key(filesize.ParamErrorLines), "must be above %s (%d)",
			filesize.ParamMaxLines, soft)
	}
}

func (val *validator) limitParam(
	at *Field, params map[string]any, key string, floor int,
) int {
	raw, set := params[key]
	if !set {
		return 0
	}
	number, ok := raw.(int)
	if !ok || number < floor {
		val.fail(at.Key(key), "must be an integer of at least %d", floor)
		return 0
	}
	return number
}
