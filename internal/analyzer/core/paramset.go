package core

import (
	"slices"
	"strconv"
)

// ParamSet is a typed view over one analyzer's raw parameter map. A preset
// literal is a Go int, a YAML config decodes numbers as float64, and a
// project config file might supply a string for the same key, so each
// accessor coerces every shape a value could arrive in before falling back
// to the default.
type ParamSet map[string]any

// Int returns the positive integer value of key, coercing int, int64,
// float64, and numeric-string representations. It returns fallback when key
// is absent, not coercible, or zero/negative, since every current caller uses
// Int for a threshold that is meaningless at zero or below.
func (p ParamSet) Int(key string, fallback int) int {
	value, ok := 0, false
	switch v := p[key].(type) {
	case int:
		value, ok = v, true
	case int64:
		value, ok = int(v), true
	case float64:
		value, ok = int(v), true
	case string:
		parsed, err := strconv.Atoi(v)
		value, ok = parsed, err == nil
	}
	if !ok || value <= 0 {
		return fallback
	}
	return value
}

// Bool returns the boolean value of key, coercing a bool or a parseable
// string, or fallback when key is absent or not coercible.
func (p ParamSet) Bool(key string, fallback bool) bool {
	switch v := p[key].(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

// String returns the string value of key, or fallback when key is absent or
// not a string.
func (p ParamSet) String(key string, fallback string) string {
	v, ok := p[key].(string)
	if !ok {
		return fallback
	}
	return v
}

// severityByName is the vocabulary a configured severity is spelled in. It is
// the single source for both reading a value and rejecting a bad one, so a
// caller cannot validate against a list that drifts from what this accepts.
var severityByName = map[string]Severity{
	"error":   SeverityError,
	"warning": SeverityWarning,
	"info":    SeverityInfo,
}

// SeverityNames lists the accepted spellings, sorted for a stable diagnostic.
func SeverityNames() []string {
	names := make([]string, 0, len(severityByName))
	for name := range severityByName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ValidSeverityName reports whether name is a severity ParamSet.Severity reads.
// why: an unrecognized value falls back silently, which would turn a typo in a
// flag into a policy the caller never chose.
func ValidSeverityName(name string) bool {
	_, ok := severityByName[name]
	return ok
}

// Severity returns the Severity named by the conventional "severity" key, or
// fallback when the key is absent or names nothing recognized.
// why: severity is policy, not a property of the defect - the same finding blocks
// one project and only advises another - so a configurable analyzer reads it here
// instead of freezing it into a package-level reporter.
func (p ParamSet) Severity(fallback Severity) Severity {
	if severity, ok := severityByName[p.String("severity", "")]; ok {
		return severity
	}
	return fallback
}

// Params returns the ParamSet for analyzerID. It is nil (safe to call
// Int/Bool/String on) when no config, or no per-analyzer params, exist.
func (ctx *AnalysisContext) Params(analyzerID string) ParamSet {
	if ctx.Config == nil {
		return nil
	}
	cfg, ok := ctx.Config.Analyzers[analyzerID]
	if !ok {
		return nil
	}
	return cfg.Params
}
