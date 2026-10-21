package config

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
)

func isEnabled(cfg *GovernanceConfig, analyzerID string) bool {
	ac, ok := cfg.Analyzers[analyzerID]
	if !ok {
		return true
	}
	return ac.Enabled
}

func TestSetParam(t *testing.T) {
	t.Parallel()

	const analyzerID, key, value = "golangci-lint", "config_path", "./golangci.yml"
	testCases := []struct {
		name   string
		seeded map[string]any
	}{
		{name: "creates the entry when the analyzer has none"},
		{
			name:   "overwrites an entry without dropping its other params",
			seeded: map[string]any{"timeout": 30},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := Defaults()
			if testCase.seeded != nil {
				cfg.Analyzers[analyzerID] = AnalyzerConfig{Params: testCase.seeded}
			}
			cfg.SetParam(analyzerID, key, value)

			if got := cfg.Analyzers[analyzerID].Params[key]; got != value {
				t.Errorf("Param(%q) = %v, want %v", key, got, value)
			}
			for otherKey, want := range testCase.seeded {
				if got := cfg.Analyzers[analyzerID].Params[otherKey]; got != want {
					t.Errorf("Param(%q) = %v, want %v", otherKey, got, want)
				}
			}
		})
	}
}

func effectiveSeverity(cfg *GovernanceConfig, analyzerID string, from finding.Severity) string {
	probe := []finding.Finding{{AnalyzerID: analyzerID, Severity: from}}
	return overrides.Apply(probe, analyzerID, cfg.SeverityRules)[0].Severity.String()
}

func TestSetSeverity(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.SetSeverity("typographic-markers", "error")
	cfg.SetSeverity("doc-comment", "info")
	cfg.SetSeverity("typographic-markers", "warning")

	testCases := []struct {
		analyzerID string
		want       string
	}{
		{"typographic-markers", "warning"},
		{"doc-comment", "info"},
		{"naming", "error"},
	}
	for _, testCase := range testCases {
		got := effectiveSeverity(cfg, testCase.analyzerID, finding.SeverityError)
		if got != testCase.want {
			t.Errorf("severity(%s) = %q, want %q", testCase.analyzerID, got, testCase.want)
		}
	}
	for _, rule := range cfg.SeverityRules {
		if !rule.Files.IsOpen() {
			t.Errorf("severity rule is scoped to files: %+v", rule)
		}
	}
}
