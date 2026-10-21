package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

type stubAnalyzer struct {
	findings []finding.Finding
}

func (stub stubAnalyzer) Analyze(_ *pass.Context) []finding.Finding {
	return stub.findings
}

func TestPipelineRun(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "a1", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "a1", Message: "issue 1", Severity: finding.SeverityWarning},
			},
		}},
		{ID: "a2", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "a2", Message: "issue 2", Severity: finding.SeverityError},
			},
		}},
	})

	ctx := &pass.Context{}
	findings := pipe.run(ctx)
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

func TestPipelineDisabledAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := &pass.AnalysisConfig{
		Analyzers: map[string]pass.AnalyzerConfig{
			"disabled": {Enabled: false},
		},
	}
	pipe := newPipeline(cfg, []registry.Built{
		{ID: "disabled", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
		}},
		{ID: "enabled", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
		}},
	})

	ctx := &pass.Context{}
	findings := pipe.run(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

type stubInPlaceAnalyzer struct{}

func (stub stubInPlaceAnalyzer) TransformsInPlace() {}

func (stub stubInPlaceAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if len(ctx.Findings) > 0 {
		ctx.Findings = ctx.Findings[1:]
	}
	return []finding.Finding{{AnalyzerID: "should-not-be-appended"}}
}

var _ pass.InPlaceAnalyzer = stubInPlaceAnalyzer{}

func TestPipelineInPlaceAnalyzerNotAppended(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "producer", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "one"}, {AnalyzerID: "two"},
			},
		}},
		{ID: "transformer", Analyzer: stubInPlaceAnalyzer{}},
	})

	findings := pipe.run(&pass.Context{})
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

func TestPipelineRunsInRegistrationOrder(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "ast", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "ast"}},
		}},
		{ID: "static", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "static"}},
		}},
		{ID: "arch", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "arch"}},
		}},
	})

	ctx := &pass.Context{}
	findings := pipe.run(ctx)

	want := []string{"ast", "static", "arch"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %d: %+v", len(want), len(findings), findings)
	}
	for idx, id := range want {
		if findings[idx].AnalyzerID != id {
			t.Errorf(
				"findings[%d].AnalyzerID = %q, want %q (registration order violated)",
				idx,
				findings[idx].AnalyzerID,
				id,
			)
		}
	}
}

func TestPipelineAppliesOverrides(t *testing.T) {
	t.Parallel()

	cfg := toAnalysisConfig(&config.GovernanceConfig{
		Severities: map[string]string{"golangci-lint": "info"},
		Overrides: []pass.Override{{
			Files:     glob.CompileSet([]string{"web/test/**"}),
			Analyzers: map[string]pass.OverrideConfig{"file-size": {Severity: "warning"}},
		}},
	})
	pipe := newPipeline(cfg, []registry.Built{
		{ID: "golangci-lint", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{
					AnalyzerID: "golangci-lint/errcheck",
					File:       "main.go",
					Severity:   finding.SeverityError,
				},
			},
		}},
		{ID: "file-size", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityError},
				{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: finding.SeverityError},
			},
		}},
	})

	findings := pipe.run(&pass.Context{})
	want := []finding.Severity{finding.SeverityInfo, finding.SeverityWarning, finding.SeverityError}
	for idx, severity := range want {
		if findings[idx].Severity != severity {
			t.Errorf("finding %d severity = %v, want %v", idx, findings[idx].Severity, severity)
		}
	}
}
