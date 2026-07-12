package output

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestOutputAnalyzer_SortBySeverity(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	ctx := &pass.Context{
		Findings: []finding.Finding{
			{
				AnalyzerID: "a",
				Severity:   finding.SeverityInfo,
				File:       "b.go",
				Line:       5,
				Message:    "info",
			},
			{
				AnalyzerID: "b",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "error",
			},
			{
				AnalyzerID: "c",
				Severity:   finding.SeverityWarning,
				File:       "c.go",
				Line:       1,
				Message:    "warn",
			},
		},
	}

	a.Analyze(ctx)

	if len(ctx.Findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(ctx.Findings))
	}
	if ctx.Findings[0].Severity != finding.SeverityError {
		t.Errorf("first should be error, got %d", ctx.Findings[0].Severity)
	}
	if ctx.Findings[1].Severity != finding.SeverityWarning {
		t.Errorf("second should be warning, got %d", ctx.Findings[1].Severity)
	}
	if ctx.Findings[2].Severity != finding.SeverityInfo {
		t.Errorf("third should be info, got %d", ctx.Findings[2].Severity)
	}
}

func TestOutputAnalyzer_SortByFileThenLine(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	ctx := &pass.Context{
		Findings: []finding.Finding{
			{AnalyzerID: "a", Severity: finding.SeverityWarning, File: "b.go", Line: 5},
			{AnalyzerID: "b", Severity: finding.SeverityWarning, File: "a.go", Line: 10},
			{AnalyzerID: "c", Severity: finding.SeverityWarning, File: "a.go", Line: 3},
		},
	}

	a.Analyze(ctx)

	if ctx.Findings[0].File != "a.go" || ctx.Findings[0].Line != 3 {
		t.Errorf("expected a.go:3 first, got %s:%d", ctx.Findings[0].File, ctx.Findings[0].Line)
	}
	if ctx.Findings[1].File != "a.go" || ctx.Findings[1].Line != 10 {
		t.Errorf("expected a.go:10 second, got %s:%d", ctx.Findings[1].File, ctx.Findings[1].Line)
	}
	if ctx.Findings[2].File != "b.go" || ctx.Findings[2].Line != 5 {
		t.Errorf("expected b.go:5 third, got %s:%d", ctx.Findings[2].File, ctx.Findings[2].Line)
	}
}

func TestOutputAnalyzer_Dedup(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	ctx := &pass.Context{
		Findings: []finding.Finding{
			{
				AnalyzerID: "x",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       1,
				Message:    "dup",
			},
			{
				AnalyzerID: "x",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       1,
				Message:    "dup",
			},
			{
				AnalyzerID: "x",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       1,
				Message:    "dup",
			},
			{
				AnalyzerID: "y",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       2,
				Message:    "other",
			},
		},
	}

	a.Analyze(ctx)

	if len(ctx.Findings) != 2 {
		t.Fatalf("expected 2 after dedup, got %d", len(ctx.Findings))
	}
}

func TestOutputAnalyzer_Empty(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	ctx := &pass.Context{Findings: nil}
	a.Analyze(ctx)
	if len(ctx.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(ctx.Findings))
	}
}

func TestOutputAnalyzer_SortByLineWithinOneFile(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	ctx := &pass.Context{
		Findings: []finding.Finding{
			{AnalyzerID: "a", Severity: finding.SeverityWarning, File: "a.go", Line: 9},
			{AnalyzerID: "b", Severity: finding.SeverityWarning, File: "a.go", Line: 2},
			{AnalyzerID: "c", Severity: finding.SeverityWarning, File: "a.go", Line: 2},
		},
	}

	analyzer.Analyze(ctx)

	got := make([]string, 0, len(ctx.Findings))
	for _, finding := range ctx.Findings {
		got = append(got, finding.AnalyzerID)
	}
	want := []string{"b", "c", "a"}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}
