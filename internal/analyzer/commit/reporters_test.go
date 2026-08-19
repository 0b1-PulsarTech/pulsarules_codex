package commit

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

// TestRuleReporters_ResolvedHonoursConfiguredSeverity asserts a run's
// "severity" param for a sub-rule id overrides its compiled-in default,
// proving rules.go's 14 reporters resolve per run instead of staying
// frozen at their package-level SeverityError default.
func TestRuleReporters_ResolvedHonoursConfiguredSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *core.AnalysisContext
		want core.Severity
	}{
		{
			name: "no config keeps the compiled-in default",
			ctx:  &core.AnalysisContext{},
			want: core.SeverityError,
		},
		{
			name: "a configured severity for the sub-rule id overrides it",
			ctx: &core.AnalysisContext{Config: &core.AnalysisConfig{
				Analyzers: map[string]core.AnalyzerConfig{
					"commit-desc-required": {Params: map[string]any{"severity": "warning"}},
				},
			}},
			want: core.SeverityWarning,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reporters := defaultRuleReporters().resolved(testCase.ctx)
			msg := commitmsg.Parse(":wrench: feat(goscan):")
			findings := Validate(msg, DefaultRuleConfig(), reporters)
			if len(findings) == 0 {
				t.Fatal("expected a commit-desc-required finding")
			}
			if findings[0].Severity != testCase.want {
				t.Fatalf("Severity = %v, want %v", findings[0].Severity, testCase.want)
			}
		})
	}
}
