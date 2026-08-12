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

	check := func(t *testing.T, i int, wantRuleID string, wantSummary bool) {
		t.Helper()
		f := findings[i]
		if f.RuleID != wantRuleID {
			t.Errorf("finding %d: got RuleID=%q, want %q", i, f.RuleID, wantRuleID)
		}
		hasSummary := f.RuleSummary != ""
		if hasSummary != wantSummary {
			t.Errorf(
				"finding %d: hasSummary=%v, want %v (summary length %d)",
				i,
				hasSummary,
				wantSummary,
				len(f.RuleSummary),
			)
		}
	}

	check(t, 0, "effective-go", true)
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

	a := NewAnalyzer(nil)
	if a.Stage() != core.StageRuleInjection {
		t.Fatalf("expected StageRuleInjection, got %d", a.Stage())
	}
	if a.ID() != "rule-injection" {
		t.Fatalf("expected ID rule-injection, got %q", a.ID())
	}
}

func TestRuleInjectionAnalyzer_AnalyzePopulatesSummaries(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	a := NewAnalyzer(idx)
	ctx := &core.AnalysisContext{
		Findings: []core.Finding{
			{AnalyzerID: "naming", Severity: core.SeverityWarning, Message: "bad name"},
		},
	}

	findings := a.Analyze(ctx)
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
