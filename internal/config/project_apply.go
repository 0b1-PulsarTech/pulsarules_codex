package config

import (
	"maps"
	"slices"
)

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
