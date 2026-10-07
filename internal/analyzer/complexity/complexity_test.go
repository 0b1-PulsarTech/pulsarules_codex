package complexity

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *pass.Context
		expect int
	}{
		{
			name: "no cache",
			ctx:  &pass.Context{},
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ASTCache:     astcache.New(),
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

func TestAnalyze_HonoursConfiguredThresholds(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	cache, path := fourLineFixture(t)

	ctx := &pass.Context{
		ASTCache:     cache,
		Syntax:       summaries(cache, path),
		ChangedFiles: []pass.FileChange{{Path: path, Extension: ".go"}},
	}
	if got := len(analyzer.Analyze(ctx)); got != 0 {
		t.Fatalf("expected 0 findings against the default threshold, got %d", got)
	}

	ctx.Config = &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
		"complexity": {Params: map[string]any{"max_func_lines": 1}},
	}}
	got := analyzer.Analyze(ctx)
	if !hasFuncLinesFinding(got) {
		t.Fatalf("a configured max_func_lines=1 should report the 4-line func, got %+v", got)
	}
}

func TestAnalyze_HonoursConfiguredSeverity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	cache, path := fourLineFixture(t)
	changedFiles := []pass.FileChange{{Path: path, Extension: ".go"}}

	testCases := []struct {
		name   string
		config *pass.AnalysisConfig
		want   pass.Severity
	}{
		{
			name: "default severity is warning",
			config: &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
				"complexity": {Params: map[string]any{"max_func_lines": 1}},
			}},
			want: pass.SeverityWarning,
		},
		{
			name: "configured severity overrides the default",
			config: &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
				"complexity": {
					Params: map[string]any{"max_func_lines": 1, "severity": "error"},
				},
			}},
			want: pass.SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(&pass.Context{
				ASTCache:     cache,
				Syntax:       summaries(cache, path),
				ChangedFiles: changedFiles,
				Config:       testCase.config,
			})
			if !hasFuncLinesFinding(got) {
				t.Fatalf("expected a func-lines finding, got %+v", got)
			}
			if got[0].Severity != testCase.want {
				t.Fatalf("Severity = %v, want %v", got[0].Severity, testCase.want)
			}
		})
	}
}

func fourLineFixture(t *testing.T) (*astcache.Cache, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "foo.go")
	src := []byte("package foo\nfunc f() {\n\tx := 1\n\t_ = x\n}\n")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	cache := astcache.New()
	if _, err := cache.Parse(path, src); err != nil {
		t.Fatalf("parse source: %v", err)
	}
	return cache, path
}

func hasFuncLinesFinding(findings []pass.Finding) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Message, "is 4 lines, max 1") {
			return true
		}
	}
	return false
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	th := thresholds{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
		warnReporter:  analyzer.warnReporter,
		infoReporter:  analyzer.infoReporter,
	}

	for _, testCase := range checkFileTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cache, file := parseSourceFile(t, testCase.source)
			fc := pass.FileChange{Path: "foo.go", Extension: ".go"}
			path := cache.FileSet().Position(file.Pos()).Filename
			summary := summaries(cache, path)
			neutral, _ := summary.Get(path)
			got := append(th.checkFile(fc, neutral), th.checkGoFile(cache.FileSet(), fc, file)...)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

type checkFileTestCase struct {
	name   string
	source string
	expect int
}

func checkFileTestCases() []checkFileTestCase {
	cases := checkFileCasesThresholds()
	cases = append(cases, checkFileCasesFlagArgs()...)
	cases = append(cases, checkFileCasesMagicNumbers()...)
	return cases
}

func checkFileCasesThresholds() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "simple func no findings",
			source: "package foo\nfunc f() { x := 1; _ = x }\n",
			expect: 0,
		},
		{
			name: "flag argument detected",
			source: "package foo\n" +
				"func f(verbose bool) {\n" +
				"	_ = verbose\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "many params detected",
			source: "package foo\n" +
				"func f(a, b, c, d, e, f2 int) {\n" +
				"	_ = a; _ = b; _ = c; _ = d; _ = e; _ = f2\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "complexity high",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"	if true {}\n" +
				"}\n",
			expect: 1,
		},
	}
}

func checkFileCasesFlagArgs() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "fallback bool parameter not flagged as a flag argument",
			source: "package foo\n" +
				"func f(key string, fallback bool) bool {\n" +
				"	_ = key\n" +
				"	return fallback\n" +
				"}\n",
			expect: 0,
		},
	}
}

func summaries(cache *astcache.Cache, path string) *syntax.Cache {
	read := func(string) ([]byte, bool) { return nil, true }
	return syntax.NewCache(
		context.Background(),
		[]syntax.Provider{gosyntax.New(cache)},
		syntax.Corpus{Sources: []syntax.Source{{Path: path}}, Read: read},
	)
}

func parseSourceFile(t *testing.T, source string) (*astcache.Cache, *ast.File) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cache := astcache.New()
	file, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}
	return cache, file
}

type summaryCase struct {
	name string
	fn   syntax.Function
	test bool
	want []string
}

func summaryCases() []summaryCase {
	return []summaryCase{
		{name: "within limits", fn: syntax.Function{Name: "ok", BodyStart: 1, BodyEnd: 5}},
		{
			name: "every threshold breached",
			fn: syntax.Function{
				Name: "big", Line: 3, BodyStart: 3, BodyEnd: 100, Params: 6, Complexity: 16,
			},
			want: []string{
				"big has cyclomatic complexity 16, max 15",
				"big is 98 lines, max 80",
				"big has 6 parameters, max 5",
			},
		},
		{
			name: "test file relaxes only the length",
			fn:   syntax.Function{Name: "tbl", BodyStart: 1, BodyEnd: 100},
			test: true,
		},
		{
			name: "nested function is skipped",
			fn:   syntax.Function{Name: "lit", BodyStart: 1, BodyEnd: 200, Nested: true},
		},
		{
			name: "test container skips only the length",
			fn:   syntax.Function{Name: "main", BodyStart: 1, BodyEnd: 200, TestContainer: true},
		},
		{
			name: "test container keeps complexity and params",
			fn: syntax.Function{
				Name: "main", BodyStart: 1, BodyEnd: 200, TestContainer: true,
				Params: 6, Complexity: 16,
			},
			want: []string{
				"main has cyclomatic complexity 16, max 15",
				"main has 6 parameters, max 5",
			},
		},
	}
}

func TestCheckFile_SummaryFromAnyLanguage(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	th := thresholds{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
		warnReporter:  analyzer.warnReporter,
		infoReporter:  analyzer.infoReporter,
	}
	for _, testCase := range summaryCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fc := pass.FileChange{Path: "a.dart", IsTest: testCase.test}
			got := th.checkFile(fc, &syntax.File{Functions: []syntax.Function{testCase.fn}})
			messages := make([]string, 0, len(got))
			for _, finding := range got {
				messages = append(messages, finding.Message)
			}
			if !slices.Equal(messages, testCase.want) {
				t.Fatalf("messages = %q, want %q", messages, testCase.want)
			}
		})
	}
}
