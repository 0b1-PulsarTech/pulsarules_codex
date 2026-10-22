package cli

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
)

func TestGovernanceConfig_TypographicSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
		want     string
	}{
		{name: "unset keeps the analyzer default", want: "error"},
		{name: "warning reaches the analyzer", severity: "warning", want: "warning"},
		{name: "info reaches the analyzer", severity: "info", want: "info"},
		{name: "a typo is rejected", severity: "fatal", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			opts := &version.Options{
				Tuning: flags.Tuning{TypographicSeverity: testCase.severity},
			}
			cfg, err := governance.GovernanceConfig(opts, "")
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("governanceConfig(%q) err = nil, want an error", testCase.severity)
				}
				return
			}
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := effectiveSeverity(cfg, "typographic-markers"); got != testCase.want {
				t.Errorf("severity = %q, want %q", got, testCase.want)
			}
		})
	}
}

func effectiveSeverity(cfg *config.GovernanceConfig, analyzerID string) string {
	probe := []finding.Finding{{AnalyzerID: analyzerID, Severity: finding.SeverityError}}
	rules := slices.Concat(cfg.SeverityRules, cfg.Overrides)
	return overrides.Apply(probe, analyzerID, rules)[0].Severity.String()
}
