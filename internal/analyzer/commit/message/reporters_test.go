package message

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func TestRuleSpecs_IDsNonEmptyAndDistinct(t *testing.T) {
	t.Parallel()

	seen := map[string]rule{}
	for index, spec := range ruleSpecs() {
		if spec.id == "" {
			t.Errorf("rule %d has an empty id", index)
		}
		if first, dup := seen[spec.id]; dup {
			t.Errorf("rule %d repeats id %q of rule %d", index, spec.id, first)
		}
		seen[spec.id] = rule(index)
	}
}

func TestDefaultRuleReporters_Severity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id   string
		want finding.Severity
	}{
		{"commit-initial", finding.SeverityError},
		{"commit-merge", finding.SeverityError},
		{"commit-emoji-required", finding.SeverityError},
		{"commit-emoji-count", finding.SeverityError},
		{"commit-type-required", finding.SeverityError},
		{"commit-type-enum", finding.SeverityError},
		{"commit-scope-charset", finding.SeverityError},
		{"commit-desc-required", finding.SeverityError},
		{"commit-desc-length", finding.SeverityError},
		{"commit-desc-capitalize", finding.SeverityError},
		{"commit-desc-no-period", finding.SeverityError},
		{"commit-body-length", finding.SeverityError},
		{"commit-body-total-length", finding.SeverityError},
		{"commit-no-coauthor", finding.SeverityError},
		{"commit-emoji-prohibited", finding.SeverityError},
		{"commit-emoji-catalog", finding.SeverityError},
		{"commit-emoji-repeat", finding.SeverityError},
		{"commit-emoji-soft-repeat", finding.SeverityInfo},
	}
	reporters := defaultRuleReporters()
	if len(testCases) != len(reporters) {
		t.Fatalf("%d cases for %d rules", len(testCases), len(reporters))
	}
	for index, testCase := range testCases {
		got := reporters[index].New("probe")
		if got.AnalyzerID != testCase.id || got.Severity != testCase.want {
			t.Errorf(
				"rule %d = %s at %v, want %s at %v",
				index, got.AnalyzerID, got.Severity, testCase.id, testCase.want,
			)
		}
	}
}
