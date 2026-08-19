package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type stubAnalyzer struct {
	id       string
	stage    pass.StageID
	findings []pass.Finding
}

func (s stubAnalyzer) ID() string          { return s.id }
func (s stubAnalyzer) Stage() pass.StageID { return s.stage }
func (s stubAnalyzer) Analyze(_ *pass.Context) []pass.Finding {
	return s.findings
}

func TestStageRunnerRegisterAndRun(t *testing.T) {
	t.Parallel()

	r := newPipeline(nil)
	r.Register(stubAnalyzer{
		id:    "a1",
		stage: pass.StageStatic,
		findings: []pass.Finding{
			{AnalyzerID: "a1", Message: "issue 1", Severity: pass.SeverityWarning},
		},
	})
	r.Register(stubAnalyzer{
		id:    "a2",
		stage: pass.StageAST,
		findings: []pass.Finding{
			{AnalyzerID: "a2", Message: "issue 2", Severity: pass.SeverityError},
		},
	})

	ctx := &pass.Context{}
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

	cfg := &pass.AnalysisConfig{
		Analyzers: map[string]pass.AnalyzerConfig{
			"disabled": {Enabled: false},
		},
	}
	r := newPipeline(cfg)
	r.Register(stubAnalyzer{
		id:       "disabled",
		stage:    pass.StageStatic,
		findings: []pass.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
	})
	r.Register(stubAnalyzer{
		id:       "enabled",
		stage:    pass.StageStatic,
		findings: []pass.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
	})

	ctx := &pass.Context{}
	findings := r.RunStages(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

type stubInPlaceAnalyzer struct {
	id    string
	stage pass.StageID
}

func (s stubInPlaceAnalyzer) ID() string          { return s.id }
func (s stubInPlaceAnalyzer) Stage() pass.StageID { return s.stage }
func (s stubInPlaceAnalyzer) TransformsInPlace()  {}

func (s stubInPlaceAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if len(ctx.Findings) > 0 {
		ctx.Findings = ctx.Findings[1:]
	}
	return []pass.Finding{{AnalyzerID: "should-not-be-appended"}}
}

var _ pass.InPlaceAnalyzer = stubInPlaceAnalyzer{}

func TestStageRunnerInPlaceAnalyzerNotAppended(t *testing.T) {
	t.Parallel()

	r := newPipeline(nil)
	r.Register(stubAnalyzer{
		id:    "producer",
		stage: pass.StageStatic,
		findings: []pass.Finding{
			{AnalyzerID: "one"}, {AnalyzerID: "two"},
		},
	})
	r.Register(stubInPlaceAnalyzer{id: "transformer", stage: pass.StageOutput})

	findings := r.RunStages(&pass.Context{})
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

func TestStageRunnerOrdersByStage(t *testing.T) {
	t.Parallel()

	r := newPipeline(nil)
	r.Register(stubAnalyzer{
		id:       "ast",
		stage:    pass.StageAST,
		findings: []pass.Finding{{AnalyzerID: "ast"}},
	})
	r.Register(stubAnalyzer{
		id:       "static",
		stage:    pass.StageStatic,
		findings: []pass.Finding{{AnalyzerID: "static"}},
	})
	r.Register(stubAnalyzer{
		id:       "arch",
		stage:    pass.StageArch,
		findings: []pass.Finding{{AnalyzerID: "arch"}},
	})

	ctx := &pass.Context{}
	findings := r.RunStages(ctx)

	want := []string{"static", "ast", "arch"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %d: %+v", len(want), len(findings), findings)
	}
	for idx, id := range want {
		if findings[idx].AnalyzerID != id {
			t.Errorf(
				"findings[%d].AnalyzerID = %q, want %q (stage order violated)",
				idx,
				findings[idx].AnalyzerID,
				id,
			)
		}
	}
}
