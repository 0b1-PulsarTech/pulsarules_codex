package rulesummary

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestAttach_Summaries(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []finding.Finding{
		{AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "file too long"},
		{AnalyzerID: "naming", Severity: finding.SeverityWarning, Message: "bad name"},
		{AnalyzerID: "unknown-analyzer", Severity: finding.SeverityError, Message: "unknown"},
		{
			AnalyzerID: "golangci-lint/gocyclo",
			Severity:   finding.SeverityWarning,
			Message:    "high cyclomatic",
		},
		{
			AnalyzerID: "commit-lint",
			Severity:   finding.SeverityError,
			Message:    "type required",
		},
		{
			AnalyzerID: "arch-boundary",
			Severity:   finding.SeverityError,
			Message:    "boundary violation",
		},
	}

	Attach(idx, testRules())(findings)

	check := func(t *testing.T, findingIdx int, wantRuleID string, wantSummary bool) {
		t.Helper()
		finding := findings[findingIdx]
		if finding.RuleID != wantRuleID {
			t.Errorf("finding %d: got RuleID=%q, want %q", findingIdx, finding.RuleID, wantRuleID)
		}
		hasSummary := finding.RuleSummary != ""
		if hasSummary != wantSummary {
			t.Errorf(
				"finding %d: hasSummary=%v, want %v (summary length %d)",
				findingIdx,
				hasSummary,
				wantSummary,
				len(finding.RuleSummary),
			)
		}
	}

	check(t, 0, "source-files", true)
	check(t, 1, "naming", true)
	check(t, 2, "", false)
	check(t, 3, "code-smells", true)
	check(t, 4, "commits", true)
	check(t, 5, "dependency-rule", true)
}

func TestAttach_NilIndex(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "file too long"},
	}

	Attach(nil, testRules())(findings)

	if findings[0].RuleSummary != "" {
		t.Fatalf("expected empty RuleSummary with nil index, got %q", findings[0].RuleSummary)
	}
}

func TestAttach_PreexistingRuleID(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []finding.Finding{
		{
			AnalyzerID: "some-analyzer",
			Severity:   finding.SeverityWarning,
			Message:    "msg",
			RuleID:     "imports",
		},
	}

	Attach(idx, testRules())(findings)

	if findings[0].RuleID != "imports" {
		t.Fatalf("RuleID should remain pre-set: got %q", findings[0].RuleID)
	}
	if len(findings[0].RuleSummary) == 0 {
		t.Fatal("expected RuleSummary to be populated from pre-set RuleID")
	}
}

func TestAttach_AlreadyHasSummary(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []finding.Finding{
		{
			AnalyzerID:  "file-size",
			Severity:    finding.SeverityWarning,
			Message:     "msg",
			RuleSummary: "custom summary",
		},
	}

	Attach(idx, testRules())(findings)

	if findings[0].RuleSummary != "custom summary" {
		t.Fatalf("existing RuleSummary should not be overwritten: got %q", findings[0].RuleSummary)
	}
}

func TestAttach(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	testCases := []struct {
		name     string
		index    *knowledge.Index
		defaults map[string]string
		wantRule string
	}{
		{
			"rule comes from the handed map",
			idx,
			map[string]string{"naming": "comments"},
			"comments",
		},
		{"analyzer missing from the map gets none", idx, map[string]string{}, ""},
		{"nil index attaches nothing", nil, map[string]string{"naming": "naming"}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			findings := []finding.Finding{{AnalyzerID: "naming", Message: "bad name"}}

			got := Attach(testCase.index, testCase.defaults)(findings)

			if len(got) != 1 || got[0].RuleID != testCase.wantRule {
				t.Fatalf("findings = %+v, want RuleID %q", got, testCase.wantRule)
			}
			if hasSummary := got[0].RuleSummary != ""; hasSummary != (testCase.wantRule != "") {
				t.Fatalf("summary present = %v for rule %q", hasSummary, testCase.wantRule)
			}
		})
	}
}

func TestLookupRule(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	rules := testRules()
	testCases := []struct {
		analyzerID string
		wantRuleID string
	}{
		{"top-of-file", "comments"},
		{"big-comment", "comments"},
		{"history-narration", "comments"},
		{"test-comment", "comments"},
		{"doc-comment", "effective-go"},
		{"body-header-comment", "effective-go"},
		{"too-many-results", "effective-go"},
		{"ungrouped-interfaces", "effective-go"},
		{"single-letter-name", "naming"},
		{"variadic-index", "flag-arguments"},
		{"package-level-var", "code-smells"},
		{"package-file-count", "code-placement-go"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.analyzerID, func(t *testing.T) {
			t.Parallel()

			if got := lookupRule(rules, testCase.analyzerID); got != testCase.wantRuleID {
				t.Fatalf(
					"lookupRule(%q) = %q, want %q",
					testCase.analyzerID,
					got,
					testCase.wantRuleID,
				)
			}
			if testCase.wantRuleID != "" &&
				idx.Summary(knowledge.KindRule, testCase.wantRuleID) == "" {
				t.Fatalf("rule %q has no summary in the knowledge base", testCase.wantRuleID)
			}
		})
	}
}
