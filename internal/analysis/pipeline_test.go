package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

type stubAnalyzer struct {
	id       string
	stage    core.StageID
	findings []core.Finding
}

func (stub stubAnalyzer) ID() string          { return stub.id }
func (stub stubAnalyzer) Stage() core.StageID { return stub.stage }
func (stub stubAnalyzer) Analyze(_ *core.AnalysisContext) []core.Finding {
	return stub.findings
}

func TestStageRunnerRegisterAndRun(t *testing.T) {
	t.Parallel()

	runner := NewStageRunner(nil)
	runner.Register(stubAnalyzer{
		id:    "a1",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "a1", Message: "issue 1", Severity: core.SeverityWarning},
		},
	})
	runner.Register(stubAnalyzer{
		id:    "a2",
		stage: core.StageAST,
		findings: []core.Finding{
			{AnalyzerID: "a2", Message: "issue 2", Severity: core.SeverityError},
		},
	})

	ctx := &core.AnalysisContext{}
	findings := runner.RunStages(ctx)
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
	runner := NewStageRunner(cfg)
	runner.Register(stubAnalyzer{
		id:       "disabled",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
	})
	runner.Register(stubAnalyzer{
		id:       "enabled",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
	})

	ctx := &core.AnalysisContext{}
	findings := runner.RunStages(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

type stubInPlaceAnalyzer struct {
	id    string
	stage core.StageID
}

func (stub stubInPlaceAnalyzer) ID() string          { return stub.id }
func (stub stubInPlaceAnalyzer) Stage() core.StageID { return stub.stage }
func (stub stubInPlaceAnalyzer) TransformsInPlace()  {}

func (stub stubInPlaceAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if len(ctx.Findings) > 0 {
		ctx.Findings = ctx.Findings[1:]
	}
	return []core.Finding{{AnalyzerID: "should-not-be-appended"}}
}

var _ core.InPlaceAnalyzer = stubInPlaceAnalyzer{}

func TestStageRunnerInPlaceAnalyzerNotAppended(t *testing.T) {
	t.Parallel()

	runner := NewStageRunner(nil)
	runner.Register(stubAnalyzer{
		id:    "producer",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "one"}, {AnalyzerID: "two"},
		},
	})
	runner.Register(stubInPlaceAnalyzer{id: "transformer", stage: core.StageOutput})

	findings := runner.RunStages(&core.AnalysisContext{})
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

	runner := NewStageRunner(nil)
	runner.Register(stubAnalyzer{
		id:       "ast",
		stage:    core.StageAST,
		findings: []core.Finding{{AnalyzerID: "ast"}},
	})
	runner.Register(stubAnalyzer{
		id:       "static",
		stage:    core.StageStatic,
		findings: []core.Finding{{AnalyzerID: "static"}},
	})
	runner.Register(stubAnalyzer{
		id:       "arch",
		stage:    core.StageArch,
		findings: []core.Finding{{AnalyzerID: "arch"}},
	})

	ctx := &core.AnalysisContext{}
	findings := runner.RunStages(ctx)

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

func TestStageRunnerAppliesOverrides(t *testing.T) {
	t.Parallel()

	cfg := toAnalysisConfig(&config.GovernanceConfig{
		Severities: map[string]string{"golangci-lint": "info"},
		Overrides: []core.Override{{
			Files:     []string{"web/test/**"},
			Analyzers: map[string]core.OverrideConfig{"file-size": {Severity: "warning"}},
		}},
	})
	runner := NewStageRunner(cfg)
	runner.Register(stubAnalyzer{
		id:    "golangci-lint",
		stage: core.StageStatic,
		findings: []core.Finding{{
			AnalyzerID: "golangci-lint/errcheck", File: "main.go", Severity: core.SeverityError,
		}},
	})
	runner.Register(stubAnalyzer{
		id:    "file-size",
		stage: core.StageStatic,
		findings: []core.Finding{
			{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: core.SeverityError},
			{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: core.SeverityError},
		},
	})

	findings := runner.RunStages(&core.AnalysisContext{})
	want := []core.Severity{core.SeverityInfo, core.SeverityWarning, core.SeverityError}
	for idx, severity := range want {
		if findings[idx].Severity != severity {
			t.Errorf("finding %d severity = %v, want %v", idx, findings[idx].Severity, severity)
		}
	}
}
