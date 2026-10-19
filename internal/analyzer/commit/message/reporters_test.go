package message

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func TestRuleReporters_ResolvedSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *pass.Context
		want finding.Severity
	}{
		{
			name: "no config keeps the compiled-in default",
			ctx:  &pass.Context{},
			want: finding.SeverityError,
		},
		{
			name: "a configured severity for the sub-rule id overrides it",
			ctx: &pass.Context{Config: &pass.AnalysisConfig{
				Analyzers: map[string]pass.AnalyzerConfig{
					"commit-desc-required": {Params: map[string]any{"severity": "warning"}},
				},
			}},
			want: finding.SeverityWarning,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reporters := defaultRuleReporters().resolved(testCase.ctx)
			msg := commitmsg.Parse(":wrench: feat(goscan):")
			findings := validate(msg, defaultRuleConfig(), reporters)
			if len(findings) == 0 {
				t.Fatal("expected a commit-desc-required finding")
			}
			if findings[0].Severity != testCase.want {
				t.Fatalf("Severity = %v, want %v", findings[0].Severity, testCase.want)
			}
		})
	}
}
