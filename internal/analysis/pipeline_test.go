package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// stubAnalyzer is a minimal Analyzer for testing the stage runner.
type stubAnalyzer struct {
	id       string
	stage    core.StageID
	findings []core.Finding
}

func (s stubAnalyzer) ID() string          { return s.id }
func (s stubAnalyzer) Stage() core.StageID { return s.stage }
func (s stubAnalyzer) Analyze(_ *core.AnalysisContext) []core.Finding {
	return s.findings
}

func TestStageRunnerRegisterAndRun(t *testing.T) {
	t.Parallel()

	r := NewStageRunner(nil)
	r.Register(stubAnalyzer{
		id:    "a1",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "a1", Message: "issue 1", Severity: core.SeverityWarning},
		},
	})
	r.Register(stubAnalyzer{
		id:    "a2",
		stage: core.StageAST,
		findings: []core.Finding{
			{AnalyzerID: "a2", Message: "issue 2", Severity: core.SeverityError},
		},
	})

	ctx := &core.AnalysisContext{}
	findings := r.RunStages(ctx)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].AnalyzerID != "a1" {
		t.Errorf("first finding should be from a1, got %s", findings[0].AnalyzerID)
	}
	if findings[1].AnalyzerID != "a2" {
		t.Errorf("second finding should be from a2, got %s", findings[1].AnalyzerID)
	}
}

func TestStageRunnerDisabledAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := &core.AnalysisConfig{
		Analyzers: map[string]core.AnalyzerConfig{
			"disabled": {Enabled: false},
		},
	}
	r := NewStageRunner(cfg)
	r.Register(stubAnalyzer{
		id:       "disabled",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
	})
	r.Register(stubAnalyzer{
		id:       "enabled",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
	})

	ctx := &core.AnalysisContext{}
	findings := r.RunStages(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

// TestStageRunnerRunsRegardlessOfContextShape pins the deletion of the
// Needs()/Requirements gate: it used to skip a registered analyzer when
// NeedsAST/NeedsGitHistory went unmet, checking ctx.ChangedFiles instead of
// ctx.ASTCache, so a nil ASTCache with non-nil ChangedFiles passed anyway.
// The gate is gone; an analyzer now runs on a bare context.
func TestStageRunnerRunsRegardlessOfContextShape(t *testing.T) {
	t.Parallel()

	r := NewStageRunner(nil)
	r.Register(stubAnalyzer{
		id:    "bare-context",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "bare-context", Message: "ran anyway"},
		},
	})

	findings := r.RunStages(&core.AnalysisContext{})
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding on a bare context, got %d: %+v", len(findings), findings)
	}
}

// stubInPlaceAnalyzer mutates ctx.Findings itself (e.g. dropping the first
// one) and returns a non-nil slice from Analyze - if RunStages still
// appended that return value on top, as it would for a plain Analyzer, the
// findings count below would be wrong.
type stubInPlaceAnalyzer struct {
	id    string
	stage core.StageID
}

func (s stubInPlaceAnalyzer) ID() string          { return s.id }
func (s stubInPlaceAnalyzer) Stage() core.StageID { return s.stage }
func (s stubInPlaceAnalyzer) TransformsInPlace()  {}

func (s stubInPlaceAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if len(ctx.Findings) > 0 {
		ctx.Findings = ctx.Findings[1:]
	}
	return []core.Finding{{AnalyzerID: "should-not-be-appended"}}
}

var _ core.InPlaceAnalyzer = stubInPlaceAnalyzer{}

// TestStageRunnerInPlaceAnalyzerNotAppended proves RunStages checks
// core.InPlaceAnalyzer instead of trusting every in-place analyzer to
// return nil: a mutating analyzer's own Analyze return value never lands
// in ctx.Findings, only the mutation it made directly.
func TestStageRunnerInPlaceAnalyzerNotAppended(t *testing.T) {
	t.Parallel()

	r := NewStageRunner(nil)
	r.Register(stubAnalyzer{
		id:    "producer",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "one"}, {AnalyzerID: "two"},
		},
	})
	r.Register(stubInPlaceAnalyzer{id: "transformer", stage: core.StageOutput})

	findings := r.RunStages(&core.AnalysisContext{})
	if len(findings) != 1 {
		t.Fatalf(
			"expected 1 finding (one dropped, none appended), got %d: %+v",
			len(findings),
			findings,
		)
	}
	if findings[0].AnalyzerID != "two" {
		t.Fatalf("expected the surviving finding to be %q, got %q", "two", findings[0].AnalyzerID)
	}
}

func TestStageRunnerStageOrder(t *testing.T) {
	t.Parallel()

	// Register out of stage order (AST, static, arch) so a pass proves
	// RunStages orders by stage, not by registration order.
	r := NewStageRunner(nil)
	r.Register(stubAnalyzer{
		id:       "ast",
		stage:    core.StageAST,
		findings: []core.Finding{{AnalyzerID: "ast"}},
	})
	r.Register(stubAnalyzer{
		id:       "static",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "static"}},
	})
	r.Register(stubAnalyzer{
		id:       "arch",
		stage:    core.StageArch,
		findings: []core.Finding{{AnalyzerID: "arch"}},
	})

	ctx := &core.AnalysisContext{}
	findings := r.RunStages(ctx)

	want := []string{"static", "ast", "arch"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %d: %+v", len(want), len(findings), findings)
	}
	for i, id := range want {
		if findings[i].AnalyzerID != id {
			t.Errorf(
				"findings[%d].AnalyzerID = %q, want %q (stage order violated)",
				i,
				findings[i].AnalyzerID,
				id,
			)
		}
	}
}
