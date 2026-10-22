package syntaxreport

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type stubProvider struct {
	parseErrors map[string]string
	fail        error
}

func (stubProvider) Languages() []language.ID {
	return []language.ID{
		language.Go, language.Dart, language.TypeScript, language.JavaScript, language.Svelte,
	}
}

func (provider stubProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	if provider.fail != nil {
		return nil, provider.fail
	}
	files := map[string]*syntax.File{}
	for _, source := range corpus.Sources {
		files[source.Path] = &syntax.File{
			Path:       source.Path,
			Language:   language.ForExtension(filepath.Ext(source.Path)).ID,
			ParseError: provider.parseErrors[source.Path],
		}
	}
	return files, nil
}

func buildContext(provider stubProvider, paths []string, generated string) *pass.Context {
	ctx := &pass.Context{}
	sources := make([]syntax.Source, 0, len(paths))
	for _, path := range paths {
		ctx.ChangedFiles = append(ctx.ChangedFiles, pass.FileChange{
			Path: path, Generated: path == generated,
		})
		sources = append(sources, syntax.Source{Path: path})
	}
	read := func(string) ([]byte, bool) { return nil, true }
	ctx.Syntax = syntax.NewCache(
		[]syntax.Provider{provider},
		syntax.Corpus{Sources: sources, Read: read})
	return ctx
}

type digest struct {
	ID       string
	Severity finding.Severity
	File     string
	Line     int
	Message  string
}

func summarise(findings []finding.Finding) []digest {
	got := make([]digest, 0, len(findings))
	for _, item := range findings {
		got = append(got, digest{
			ID: item.AnalyzerID, Severity: item.Severity, File: item.File,
			Line: item.Line, Message: item.Message,
		})
	}
	return got
}

type analyzeCase struct {
	name      string
	language  language.ID
	provider  stubProvider
	paths     []string
	generated string
	noCache   bool
	want      []digest
}

func unparsedCases() []analyzeCase {
	warn := finding.SeverityWarning
	return []analyzeCase{
		{
			name: "go file with a position in the reason", language: language.Go,
			provider: stubProvider{parseErrors: map[string]string{
				"a.go": "parse a.go: a.go:7:3: expected ';'",
			}},
			paths: []string{"a.go", "b.go"},
			want: []digest{{
				ID: "go-syntax/unparsed", Severity: warn, File: "a.go", Line: 7,
				Message: "a.go was not analysed: parse a.go: a.go:7:3: expected ';'",
			}},
		},
		{
			name: "dart file with a line in the reason", language: language.Dart,
			provider: stubProvider{parseErrors: map[string]string{"a.dart": "line 12: bad token"}},
			paths:    []string{"a.dart"},
			want: []digest{{
				ID: "dart-syntax/unparsed", Severity: warn, File: "a.dart",
				Line: 12, Message: "a.dart was not analysed: line 12: bad token",
			}},
		},
		{
			name: "typescript file without a line", language: language.TypeScript,
			provider: stubProvider{parseErrors: map[string]string{
				"a.ts": "skipped: parse exceeded the 10s time limit",
			}},
			paths: []string{"a.ts"},
			want: []digest{{
				ID: "typescript-syntax/unparsed", Severity: warn, File: "a.ts",
				Message: "a.ts was not analysed: skipped: parse exceeded the 10s time limit",
			}},
		},
		{
			name: "javascript file", language: language.JavaScript,
			provider: stubProvider{parseErrors: map[string]string{"a.mjs": "line x: bad"}},
			paths:    []string{"a.mjs"},
			want: []digest{{
				ID: "javascript-syntax/unparsed", Severity: warn, File: "a.mjs",
				Message: "a.mjs was not analysed: line x: bad",
			}},
		},
		{
			name: "svelte file", language: language.Svelte,
			provider: stubProvider{parseErrors: map[string]string{"a.svelte": "line 3: bad"}},
			paths:    []string{"a.svelte"},
			want: []digest{{
				ID: "svelte-syntax/unparsed", Severity: warn, File: "a.svelte",
				Line: 3, Message: "a.svelte was not analysed: line 3: bad",
			}},
		},
	}
}

func quietCases() []analyzeCase {
	goBad := map[string]string{"a.go": "parse a.go: a.go:1:1: bad"}
	return []analyzeCase{
		{
			name: "another language is ignored", language: language.Dart,
			provider: stubProvider{parseErrors: goBad}, paths: []string{"a.go"},
		},
		{
			name: "generated file is ignored", language: language.Go,
			provider:  stubProvider{parseErrors: map[string]string{"gen.go": "bad"}},
			paths:     []string{"gen.go"},
			generated: "gen.go",
		},
		{
			name: "clean file reports nothing", language: language.Go,
			provider: stubProvider{}, paths: []string{"a.go"},
		},
		{
			name: "unavailable for another language is ignored", language: language.Go,
			provider: stubProvider{fail: errors.New("boom")}, paths: []string{"a.dart"},
		},
		{name: "nil cache reports nothing", language: language.Go, noCache: true},
	}
}

func unavailableCases() []analyzeCase {
	return []analyzeCase{
		{
			name: "unavailable language reports one info", language: language.Dart,
			provider: stubProvider{fail: errors.New("locate dart:\n  no dart on PATH")},
			paths:    []string{"a.dart", "b.dart"},
			want: []digest{{
				ID: "dart-syntax/unavailable", Severity: finding.SeverityInfo,
				Message: "2 dart files not analysed: locate dart: no dart on PATH",
			}},
		},
		{
			name: "long reason is clipped", language: language.Dart,
			provider: stubProvider{fail: errors.New(strings.Repeat("\u00e9", 300))},
			paths:    []string{"a.dart"},
			want: []digest{{
				ID: "dart-syntax/unavailable", Severity: finding.SeverityInfo,
				Message: "1 dart files not analysed: " + strings.Repeat("\u00e9", 120),
			}},
		},
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(unparsedCases(), quietCases(), unavailableCases())
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &pass.Context{}
			if !testCase.noCache {
				ctx = buildContext(testCase.provider, testCase.paths, testCase.generated)
			}
			got := summarise(NewAnalyzer(testCase.language).Analyze(ctx))
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("findings = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		language language.ID
		wantID   string
	}{
		{language.Go, "go-syntax"},
		{language.Dart, "dart-syntax"},
		{language.TypeScript, "typescript-syntax"},
		{language.JavaScript, "javascript-syntax"},
		{language.Svelte, "svelte-syntax"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.wantID, func(t *testing.T) {
			t.Parallel()

			if got := Describe(testCase.language).ID; got != testCase.wantID {
				t.Fatalf("Describe(%q).ID = %q, want %q", testCase.language, got, testCase.wantID)
			}
		})
	}
}

func TestReasonLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		reason string
		want   int
	}{
		{name: "go position", reason: "parse a.go: a.go:7:3: expected", want: 7},
		{name: "line prefix", reason: "line 12: bad", want: 12},
		{name: "line without a number", reason: "line limit reached", want: 0},
		{name: "line number too large", reason: "line 99999999999999999999: bad", want: 0},
		{name: "no position", reason: "skipped: time limit", want: 0},
		{name: "zero line", reason: "a.go:0:1: bad", want: 0},
		{name: "empty", reason: "", want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := reasonLine(testCase.reason); got != testCase.want {
				t.Fatalf("reasonLine(%q) = %d, want %d", testCase.reason, got, testCase.want)
			}
		})
	}
}

func TestLanguages(t *testing.T) {
	t.Parallel()

	want := []language.ID{
		language.Go, language.Dart, language.TypeScript, language.JavaScript, language.Svelte,
	}
	if got := Languages(); !slices.Equal(got, want) {
		t.Fatalf("Languages() = %v, want %v", got, want)
	}
}
