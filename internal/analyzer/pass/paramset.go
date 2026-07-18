package pass

import "strconv"

type ParamSet map[string]any

func (params ParamSet) Int(key string, fallback int) int {
	value, ok := 0, false
	switch val := params[key].(type) {
	case int:
		value, ok = val, true
	case int64:
		value, ok = int(val), true
	case float64:
		value, ok = int(val), true
	case string:
		parsed, err := strconv.Atoi(val)
		value, ok = parsed, err == nil
	}
	// Callers read limits and counts; zero or below falls back to the default.
	if !ok || value <= 0 {
		return fallback
	}
	return value
}

func (params ParamSet) String(key string, fallback string) string {
	val, ok := params[key].(string)
	if !ok {
		return fallback
	}
	return val
}

func (ctx *Context) Params(analyzerID string) ParamSet {
	if ctx.Config == nil {
		return nil
	}
	cfg, ok := ctx.Config.Analyzers[analyzerID]
	if !ok {
		return nil
	}
	return cfg.Params
}
