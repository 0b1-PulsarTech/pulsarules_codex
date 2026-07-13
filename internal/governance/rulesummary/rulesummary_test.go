package rulesummary

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestInjectRuleBodies(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []finding.Finding{
		{AnalyzerID: "unknown-analyzer", Severity: finding.SeverityError, Message: "unknown"},
	}

	injectRuleBodies(findings, idx)

	if findings[0].RuleID != "" || findings[0].RuleBody != "" {
		t.Errorf(
			"unknown analyzer: got RuleID=%q and a %d-byte body, want neither",
			findings[0].RuleID,
			len(findings[0].RuleBody),
		)
	}
}

func TestInjectRuleBodies_NilIndex(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "file too long"},
	}

	injectRuleBodies(findings, nil)

	if findings[0].RuleBody != "" {
		t.Fatalf("expected empty RuleBody with nil index, got %q", findings[0].RuleBody)
	}
}

func TestInjectRuleBodies_PreexistingRuleID(t *testing.T) {
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

	injectRuleBodies(findings, idx)

	if findings[0].RuleID != "imports" {
		t.Fatalf("RuleID should remain pre-set: got %q", findings[0].RuleID)
	}
	if len(findings[0].RuleBody) == 0 {
		t.Fatal("expected RuleBody to be populated from pre-set RuleID")
	}
}

func TestInjectRuleBodies_AlreadyHasBody(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	findings := []finding.Finding{
		{
			AnalyzerID: "file-size",
			Severity:   finding.SeverityWarning,
			Message:    "msg",
			RuleBody:   "custom body",
		},
	}

	injectRuleBodies(findings, idx)

	if findings[0].RuleBody != "custom body" {
		t.Fatalf("existing RuleBody should not be overwritten: got %q", findings[0].RuleBody)
	}
}

func TestRuleInjectionAnalyzer_Stage(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer(nil)
	if a.Stage() != finding.StageRuleInjection {
		t.Fatalf("expected StageRuleInjection, got %d", a.Stage())
	}
	if a.ID() != "rule-injection" {
		t.Fatalf("expected ID rule-injection, got %q", a.ID())
	}
}

func TestRuleInjectionAnalyzer_AnalyzePopulatesBodies(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}

	a := NewAnalyzer(idx)
	ctx := &pass.Context{
		Findings: []finding.Finding{
			{
				AnalyzerID: "custom-tool",
				RuleID:     "code-smells",
				Severity:   finding.SeverityWarning,
				Message:    "high cyclomatic",
			},
		},
	}

	findings := a.Analyze(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected empty return (mutates context), got %d findings", len(findings))
	}
	if ctx.Findings[0].RuleBody == "" {
		t.Fatal("expected RuleBody to be populated via pipeline context mutation")
	}
	if ctx.Findings[0].RuleID != "code-smells" {
		t.Fatalf("expected RuleID=code-smells, got %q", ctx.Findings[0].RuleID)
	}
}
