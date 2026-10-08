package core

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

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

func severityByName() map[string]Severity {
	return map[string]Severity{
		"error":   SeverityError,
		"warning": SeverityWarning,
		"info":    SeverityInfo,
	}
}

func SeverityNames() []string {
	return slices.Sorted(maps.Keys(severityByName()))
}

func ValidateSeverityName(name string) error {
	if name == "" || ValidSeverityName(name) {
		return nil
	}
	return fmt.Errorf(
		"unknown severity %q (want %s)", name, strings.Join(SeverityNames(), "|"),
	)
}

func ValidSeverityName(name string) bool {
	_, ok := severityByName()[name]
	return ok
}

func (params ParamSet) Severity(fallback Severity) Severity {
	if severity, ok := severityByName()[params.String(ParamSeverity, "")]; ok {
		return severity
	}
	return fallback
}

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
