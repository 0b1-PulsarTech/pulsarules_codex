package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

type stubAnalyzer struct {
	stage    finding.StageID
	findings []finding.Finding
}

func (stub stubAnalyzer) Stage() finding.StageID { return stub.stage }
func (stub stubAnalyzer) Analyze(_ *pass.Context) []finding.Finding {
	return stub.findings
}

func TestStageRunnerRegisterAndRun(t *testing.T) {
	t.Parallel()

	runner := newPipeline(nil)
	runner.Register("a1", stubAnalyzer{
		stage: finding.StageStatic,
		findings: []finding.Finding{
			{AnalyzerID: "a1", Message: "issue 1", Severity: finding.SeverityWarning},
		},
	})
	runner.Register("a2", stubAnalyzer{
		stage: finding.StageAST,
		findings: []finding.Finding{
			{AnalyzerID: "a2", Message: "issue 2", Severity: finding.SeverityError},
		},
	})

	ctx := &pass.Context{}
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

	cfg := &pass.AnalysisConfig{
		Analyzers: map[string]pass.AnalyzerConfig{
			"disabled": {Enabled: false},
		},
	}
	runner := newPipeline(cfg)
	runner.Register("disabled", stubAnalyzer{
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
	})
	runner.Register("enabled", stubAnalyzer{
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
	})

	ctx := &pass.Context{}
	findings := runner.RunStages(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

type stubInPlaceAnalyzer struct {
	stage finding.StageID
}

func (stub stubInPlaceAnalyzer) Stage() finding.StageID { return stub.stage }
func (stub stubInPlaceAnalyzer) TransformsInPlace()     {}

func (stub stubInPlaceAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if len(ctx.Findings) > 0 {
		ctx.Findings = ctx.Findings[1:]
	}
	return []finding.Finding{{AnalyzerID: "should-not-be-appended"}}
}

var _ pass.InPlaceAnalyzer = stubInPlaceAnalyzer{}

func TestStageRunnerInPlaceAnalyzerNotAppended(t *testing.T) {
	t.Parallel()

	runner := newPipeline(nil)
	runner.Register("producer", stubAnalyzer{
		stage: finding.StageStatic,
		findings: []finding.Finding{
			{AnalyzerID: "one"}, {AnalyzerID: "two"},
		},
	})
	runner.Register("transformer", stubInPlaceAnalyzer{stage: finding.StageOutput})

	findings := runner.RunStages(&pass.Context{})
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

	runner := newPipeline(nil)
	runner.Register("ast", stubAnalyzer{
		stage:    finding.StageAST,
		findings: []finding.Finding{{AnalyzerID: "ast"}},
	})
	runner.Register("static", stubAnalyzer{
		stage:    finding.StageStatic,
		findings: []finding.Finding{{AnalyzerID: "static"}},
	})
	runner.Register("arch", stubAnalyzer{
		stage:    finding.StageArch,
		findings: []finding.Finding{{AnalyzerID: "arch"}},
	})

	ctx := &pass.Context{}
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
		Overrides: []pass.Override{{
			Files:     glob.CompileSet([]string{"web/test/**"}),
			Analyzers: map[string]pass.OverrideConfig{"file-size": {Severity: "warning"}},
		}},
	})
	runner := newPipeline(cfg)
	runner.Register("golangci-lint", stubAnalyzer{
		stage: finding.StageStatic,
		findings: []finding.Finding{{
			AnalyzerID: "golangci-lint/errcheck", File: "main.go", Severity: finding.SeverityError,
		}},
	})
	runner.Register("file-size", stubAnalyzer{
		stage: finding.StageStatic,
		findings: []finding.Finding{
			{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityError},
			{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: finding.SeverityError},
		},
	})

	findings := runner.RunStages(&pass.Context{})
	want := []finding.Severity{finding.SeverityInfo, finding.SeverityWarning, finding.SeverityError}
	for idx, severity := range want {
		if findings[idx].Severity != severity {
			t.Errorf("finding %d severity = %v, want %v", idx, findings[idx].Severity, severity)
		}
	}
}
