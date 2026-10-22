package projectfile

import (
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
)

func (val *validator) checkOverrides(entries []OverrideEntry) {
	field := val.root.Key("overrides")
	for idx, override := range entries {
		val.checkOverride(field.Item(idx), override)
	}
}

func (val *validator) checkOverride(field *strictyaml.Field, override OverrideEntry) {
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
	field *strictyaml.Field, id string, params map[string]any,
) {
	if params == nil {
		return
	}
	if id != filesize.ID {
		val.fail(field, "per-path params are only supported for %s, not %s",
			filesize.ID, strictyaml.ClipValue(id))
		return
	}
	val.checkAnalyzerParams(field, id, params)
	if _, set := params[pass.ParamExtensions]; set {
		val.fail(field.Key(pass.ParamExtensions), "%s is not supported per path",
			pass.ParamExtensions)
	}
}

func (val *validator) checkAnalyzerParams(
	field *strictyaml.Field, id string, params map[string]any,
) {
	val.checkParams(field, id, params)
	if id == filesize.ID {
		val.checkLimitParams(field, params)
	}
}

func (val *validator) checkLimitParams(field *strictyaml.Field, params map[string]any) {
	soft := val.limitParam(field, params, filesize.ParamMaxLines, 1)
	hard := val.limitParam(field, params, filesize.ParamErrorLines, 0)
	if soft > 0 && hard > 0 && hard <= soft {
		val.fail(field.Key(filesize.ParamErrorLines), "must be above %s (%d)",
			filesize.ParamMaxLines, soft)
	}
}

func (val *validator) limitParam(
	at *strictyaml.Field, params map[string]any, key string, floor int,
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
