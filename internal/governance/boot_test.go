package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestRegisterForScope_OutputAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.ApplyPreset()
	analysisCfg := toAnalysisConfig(cfg)

	sr := newPipeline(analysisCfg)
	sr.Register(stubAnalyzer{
		id:    "seed",
		stage: pass.StageStatic,
		findings: []pass.Finding{
			{
				AnalyzerID: "seed",
				Severity:   pass.SeverityInfo,
				File:       "b.go",
				Line:       5,
				Message:    "info",
			},
			{
				AnalyzerID: "seed",
				Severity:   pass.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   pass.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   pass.SeverityWarning,
				File:       "c.go",
				Line:       1,
				Message:    "warn",
			},
		},
	})
	sr.registerForScope(nil, nil, ScopeChanged)

	ctx := &pass.Context{Config: analysisCfg}
	findings := sr.RunStages(ctx)

	if len(findings) != 3 {
		t.Fatalf("expected 3 findings after dedup, got %d: %+v", len(findings), findings)
	}

	wantOrder := []struct {
		severity pass.Severity
		file     string
		line     int
	}{
		{pass.SeverityError, "a.go", 10},
		{pass.SeverityWarning, "c.go", 1},
		{pass.SeverityInfo, "b.go", 5},
	}
	for idx, want := range wantOrder {
		got := findings[idx]
		if got.Severity != want.severity || got.File != want.file || got.Line != want.line {
			t.Fatalf(
				"findings[%d] = %+v, want severity=%d file=%s line=%d",
				idx,
				got,
				want.severity,
				want.file,
				want.line,
			)
		}
	}
}
