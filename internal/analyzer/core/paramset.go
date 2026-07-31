package core

import "strconv"

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
