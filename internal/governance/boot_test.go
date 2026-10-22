package governance

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntaxreport"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func TestSessionTransformsSortAndDedupe(t *testing.T) {
	t.Parallel()

	seed := registry.Built{ID: "seed", Analyzer: stubAnalyzer{
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
	}}
	pipe := newPipeline(nil, []registry.Built{seed}, (&Session{}).transforms())

	findings := pipe.run(&pass.Context{}, nil)

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

func TestMaxGateFor_ReproducesScopeMembership(t *testing.T) {
	t.Parallel()

	static := []string{
		"file-size", "text-markers", "typographic-markers", "import-groups", "naming",
		"top-of-file", "big-comment", "simplification-path", "commit-lint",
		"commit-move-purity", "history-narration",
	}
	late := []string{"branch-name", "golangci-lint", "delegated-command"}
	style := []string{
		"go-syntax", "dart-syntax", "typescript-syntax", "javascript-syntax", "svelte-syntax",
		"control-flow", "shadowing", "complexity", "named-returns", "test-comment",
		"single-use-declaration", "doc-comment", "body-header-comment", "single-letter-name",
		"variadic-index", "too-many-results", "ungrouped-interfaces", "package-level-var",
		"package-file-count", "time-discipline", "context-discipline", "pprof-labels",
	}
	arch := []string{"arch-boundary", "import-cycle"}

	testCases := []struct {
		name  string
		scope Scope
		want  []string
	}{
		{
			name:  "full",
			scope: ScopeFull,
			want:  slices.Concat(static, late, style, arch),
		},
		{
			name:  "commit",
			scope: ScopeCommit,
			want:  slices.Concat(static, style),
		},
		{
			name:  "changed",
			scope: ScopeChanged,
			want:  slices.Concat(static, style, arch),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			built := registry.Analyzers(
				registry.Deps{Emoji: testCatalog(t)},
				maxGateFor(testCase.scope),
			)
			got := make([]string, len(built))
			for idx, analyzer := range built {
				got[idx] = analyzer.ID
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ids = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestDefaultProviders(t *testing.T) {
	t.Parallel()

	var got []language.ID
	for _, provider := range defaultProviders(astcache.New()) {
		got = append(got, provider.Languages()...)
	}
	want := syntaxreport.Languages()
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("provider languages = %v, syntaxreport.Languages() = %v", got, want)
	}
}

func TestSession_TypeScriptChildParser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		path   string
		source string
		wantID string
	}{
		{
			name: "single letter parameter is analysed", path: "src/a.ts",
			source: "export function run(q: number): number {\n\treturn q;\n}\n",
			wantID: "single-letter-name",
		},
		{
			name: "broken file is reported once", path: "src/broken.ts",
			source: "export function (\n", wantID: "typescript-syntax/unparsed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeFileAtDir(t, dir, testCase.path, testCase.source)
			findings, _ := analyzeTestClassifierRepo(t, dir, classifierRun{fileSet: FileSetAll})
			var got []string
			for _, finding := range findings {
				wanted := finding.AnalyzerID == testCase.wantID ||
					strings.HasSuffix(finding.AnalyzerID, "/unparsed")
				if finding.File == testCase.path && wanted {
					got = append(got, finding.AnalyzerID)
				}
			}
			if !slices.Equal(got, []string{testCase.wantID}) {
				t.Fatalf("findings for %s = %v, want [%s]", testCase.path, got, testCase.wantID)
			}
		})
	}
}
