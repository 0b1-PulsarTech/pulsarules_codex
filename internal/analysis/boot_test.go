package analysis

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestRegisterForScope_OutputAnalyzer(t *testing.T) {
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

func TestGatesFor_ReproducesScopeMembership(t *testing.T) {
	t.Parallel()

	static := []string{
		"file-size", "text-markers", "typographic-markers", "import-groups", "naming",
		"top-of-file", "big-comment", "simplification-path", "commit-lint",
		"commit-move-purity", "control-flow", "shadowing", "complexity", "named-returns",
		"vacuous-doc", "test-comment", "single-use-declaration", "time-discipline",
		"context-discipline", "pprof-labels",
	}
	tail := []string{"rule-injection", "output"}
	arch := []string{"arch-boundary", "import-cycle"}

	testCases := []struct {
		name  string
		scope Scope
		want  []string
	}{
		{
			name:  "full",
			scope: ScopeFull,
			want: slices.Concat(
				static,
				[]string{"branch-name"},
				arch,
				[]string{"golangci-lint"},
				tail,
			),
		},
		{
			name:  "commit",
			scope: ScopeCommit,
			want:  slices.Concat(static, tail),
		},
		{
			name:  "changed",
			scope: ScopeChanged,
			want:  slices.Concat(static, arch, tail),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			langs := core.NewLanguageRegistry()
			built := registry.Analyzers(nil, langs, nil, gatesFor(testCase.scope))
			got := make([]string, len(built))
			for i, a := range built {
				got[i] = a.ID()
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ids = %v, want %v", got, testCase.want)
			}
		})
	}
}
