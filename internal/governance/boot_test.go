package governance

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
)

func TestRegisterForScope_OutputAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.ApplyPreset()
	analysisCfg := toAnalysisConfig(cfg)

	sr := newPipeline(analysisCfg)
	sr.Register(stubAnalyzer{
		id:    "seed",
		stage: finding.StageStatic,
		findings: []finding.Finding{
			{
				AnalyzerID: "seed",
				Severity:   finding.SeverityInfo,
				File:       "b.go",
				Line:       5,
				Message:    "info",
			},
			{
				AnalyzerID: "seed",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   finding.SeverityError,
				File:       "a.go",
				Line:       10,
				Message:    "dup",
			},
			{
				AnalyzerID: "seed",
				Severity:   finding.SeverityWarning,
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
		severity finding.Severity
		file     string
		line     int
	}{
		{finding.SeverityError, "a.go", 10},
		{finding.SeverityWarning, "c.go", 1},
		{finding.SeverityInfo, "b.go", 5},
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
		"test-comment", "single-use-declaration", "doc-comment",
		"body-header-comment", "history-narration", "single-letter-name", "variadic-index",
		"too-many-results", "ungrouped-interfaces", "package-level-var",
		"package-file-count", "time-discipline", "context-discipline", "pprof-labels",
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
				[]string{"golangci-lint", "delegated-command"},
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

			langs := pass.NewLanguageRegistry()
			built := registry.Analyzers(nil, langs, nil, maxGateFor(testCase.scope))
			got := make([]string, len(built))
			for idx, analyzer := range built {
				got[idx] = analyzer.ID()
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ids = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestDefaultProviders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		wantLangs []syntax.Language
	}{
		{
			name:      "go and dart are registered",
			wantLangs: []syntax.Language{syntax.LangGo, syntax.LangDart},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := defaultProviders(astcache.New())
			langs := make([]syntax.Language, 0, len(got))
			for _, provider := range got {
				langs = append(langs, provider.Language())
			}
			if !slices.Equal(langs, testCase.wantLangs) {
				t.Fatalf("defaultProviders() languages = %v, want %v", langs, testCase.wantLangs)
			}
		})
	}
}
