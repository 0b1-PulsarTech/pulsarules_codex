package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type stubAnalyzer struct {
	id       string
	stage    finding.StageID
	findings []finding.Finding
}

func (s stubAnalyzer) ID() string                  { return s.id }
func (s stubAnalyzer) Stage() finding.StageID      { return s.stage }
func (s stubAnalyzer) Needs() finding.Requirements { return finding.Requirements{} }
func (s stubAnalyzer) Analyze(_ *pass.Context) []finding.Finding {
	return s.findings
}

func TestStageRunnerRegisterAndRun(t *testing.T) {
	t.Parallel()

	r := newPipeline(nil)
	r.Register(stubAnalyzer{
		id:    "a1",
		stage: finding.StageStatic,
		findings: []finding.Finding{
			{AnalyzerID: "a1", Message: "issue 1", Severity: finding.SeverityWarning},
		},
	})
	r.Register(stubAnalyzer{
		id:    "a2",
		stage: finding.StageAST,
		findings: []finding.Finding{
			{AnalyzerID: "a2", Message: "issue 2", Severity: finding.SeverityError},
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
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
	})
	r.Register(stubAnalyzer{
		id:       "enabled",
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
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

func TestStageRunnerOrdersByStage(t *testing.T) {
	t.Parallel()

	r := newPipeline(nil)
	r.Register(stubAnalyzer{
		id:       "ast",
		stage:    finding.StageAST,
		findings: []finding.Finding{{AnalyzerID: "ast"}},
	})
	r.Register(stubAnalyzer{
		id:       "static",
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "static"}},
	})
	r.Register(stubAnalyzer{
		id:       "arch",
		stage:    finding.StageArch,
		findings: []finding.Finding{{AnalyzerID: "arch"}},
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
