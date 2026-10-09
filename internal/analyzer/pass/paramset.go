package pass

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type ParamSet map[string]any

func (params ParamSet) number(key string) (int, bool) {
	switch val := params[key].(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case string:
		parsed, err := strconv.Atoi(val)
		return parsed, err == nil
	}
	return 0, false
}

func (params ParamSet) Int(key string, fallback int) int {
	value, ok := params.number(key)
	// Callers read limits and counts; zero or below falls back to the default.
	if !ok || value <= 0 {
		return fallback
	}
	return value
}

func (params ParamSet) NonNegativeInt(key string, fallback int) int {
	value, ok := params.number(key)
	// For a limit where an explicit zero means off and so must not fall back.
	if !ok || value < 0 {
		return fallback
	}
	return value
}

func (params ParamSet) Bool(key string, fallback bool) bool {
	switch val := params[key].(type) {
	case bool:
		return val
	case string:
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func (params ParamSet) Strings(key string, fallback []string) []string {
	var values []string
	switch val := params[key].(type) {
	case []string:
		values = val
	case []any:
		values = make([]string, 0, len(val))
		for _, item := range val {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	case string:
		values = strings.Split(val, ",")
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			trimmed = append(trimmed, text)
		}
	}
	// An empty configured list keeps the default instead of silently disabling the check.
	if len(trimmed) == 0 {
		return fallback
	}
	return trimmed
}

func (params ParamSet) String(key string, fallback string) string {
	val, ok := params[key].(string)
	if !ok {
		return fallback
	}
	return val
}

const ParamSeverity = "severity"

func (params ParamSet) Severity(fallback finding.Severity) finding.Severity {
	if severity, ok := finding.SeverityByName()[params.String(ParamSeverity, "")]; ok {
		return severity
	}
	return fallback
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
