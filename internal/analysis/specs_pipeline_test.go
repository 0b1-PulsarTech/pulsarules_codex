package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

// TestRegisterForScope_OutputAnalyzerSortsAndDedupes proves
// pipelineAnalyzerSpecs wires "output" into the real pipeline, not just
// isolated sorting (see output_test.go). A stub seeds unsorted duplicates
// at StageStatic with a nil repo/index, where every other analyzer
// contributes nothing - deleting the "output" entry leaves this red.
func TestRegisterForScope_OutputAnalyzerSortsAndDedupes(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.ApplyPreset()
	analysisCfg := toAnalysisConfig(cfg)

	sr := NewStageRunner(analysisCfg)
	sr.Register(stubAnalyzer{
		id:    "seed",
		stage: core.StageStatic,
		findings: []core.Finding{
			{
				AnalyzerID: "seed",
				Severity:   core.SeverityInfo,
				File:       "b.go",
				Line:       5,
				Message:    "info",
			},
			{
				AnalyzerID: "seed",
				Severity:   core.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   core.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   core.SeverityWarning,
				File:       "c.go",
				Line:       1,
				Message:    "warn",
			},
		},
	})
	sr.registerForScope(nil, nil, ScopeChanged)

	ctx := &core.AnalysisContext{Config: analysisCfg}
	findings := sr.RunStages(ctx)

	if len(findings) != 3 {
		t.Fatalf("expected 3 findings after dedup, got %d: %+v", len(findings), findings)
	}

	wantOrder := []struct {
		severity core.Severity
		file     string
		line     int
	}{
		{core.SeverityError, "a.go", 10},
		{core.SeverityWarning, "c.go", 1},
		{core.SeverityInfo, "b.go", 5},
	}
	for i, want := range wantOrder {
		got := findings[i]
		if got.Severity != want.severity || got.File != want.file || got.Line != want.line {
			t.Fatalf(
				"findings[%d] = %+v, want severity=%d file=%s line=%d",
				i,
				got,
				want.severity,
				want.file,
				want.line,
			)
		}
	}
}
