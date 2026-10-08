package ruleinjection

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestInjectRuleSummaries(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []core.Finding{
		{AnalyzerID: "file-size", Severity: core.SeverityWarning, Message: "file too long"},
		{AnalyzerID: "naming", Severity: core.SeverityWarning, Message: "bad name"},
		{AnalyzerID: "unknown-analyzer", Severity: core.SeverityError, Message: "unknown"},
		{
			AnalyzerID: "golangci-lint/gocyclo",
			Severity:   core.SeverityWarning,
			Message:    "high cyclomatic",
		},
		{
			AnalyzerID: "commit-type-required",
			Severity:   core.SeverityError,
			Message:    "type required",
		},
		{AnalyzerID: "arch-boundary", Severity: core.SeverityError, Message: "boundary violation"},
	}

	injectRuleSummaries(findings, idx)

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

func TestInjectRuleSummaries_NilIndex(t *testing.T) {
	t.Parallel()

	findings := []core.Finding{
		{AnalyzerID: "file-size", Severity: core.SeverityWarning, Message: "file too long"},
	}

	injectRuleSummaries(findings, nil)

	if findings[0].RuleSummary != "" {
		t.Fatalf("expected empty RuleSummary with nil index, got %q", findings[0].RuleSummary)
	}
}

func TestInjectRuleSummaries_PreexistingRuleID(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []core.Finding{
		{
			AnalyzerID: "some-analyzer",
			Severity:   core.SeverityWarning,
			Message:    "msg",
			RuleID:     "imports",
		},
	}

	injectRuleSummaries(findings, idx)

	if findings[0].RuleID != "imports" {
		t.Fatalf("RuleID should remain pre-set: got %q", findings[0].RuleID)
	}
	if len(findings[0].RuleSummary) == 0 {
		t.Fatal("expected RuleSummary to be populated from pre-set RuleID")
	}
}

func TestInjectRuleSummaries_AlreadyHasSummary(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []core.Finding{
		{
			AnalyzerID:  "file-size",
			Severity:    core.SeverityWarning,
			Message:     "msg",
			RuleSummary: "custom summary",
		},
	}

	injectRuleSummaries(findings, idx)

	if findings[0].RuleSummary != "custom summary" {
		t.Fatalf("existing RuleSummary should not be overwritten: got %q", findings[0].RuleSummary)
	}
}

func TestRuleInjectionAnalyzer_Stage(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.Stage() != core.StageRuleInjection {
		t.Fatalf("expected StageRuleInjection, got %d", analyzer.Stage())
	}
	if analyzer.ID() != "rule-injection" {
		t.Fatalf("expected ID rule-injection, got %q", analyzer.ID())
	}
}

func TestAnalyzer_Summaries(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	analyzer := NewAnalyzer(idx)
	ctx := &core.AnalysisContext{
		Findings: []core.Finding{
			{AnalyzerID: "naming", Severity: core.SeverityWarning, Message: "bad name"},
		},
	}

	findings := analyzer.Analyze(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected empty return (mutates context), got %d findings", len(findings))
	}
	if ctx.Findings[0].RuleSummary == "" {
		t.Fatal("expected RuleSummary to be populated via pipeline context mutation")
	}
	if ctx.Findings[0].RuleID != "naming" {
		t.Fatalf("expected RuleID=naming, got %q", ctx.Findings[0].RuleID)
	}
}

func TestLookupRule(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	rules := defaultRuleMap()
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
			if testCase.wantRuleID != "" && idx.Summary("rules", testCase.wantRuleID) == "" {
				t.Fatalf("rule %q has no summary in the knowledge base", testCase.wantRuleID)
			}
		})
	}
}
