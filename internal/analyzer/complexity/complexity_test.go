package complexity

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	src := []byte("package foo\nfunc f() {\n\tx := 1\n\t_ = x\n}\n")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	cache := astcache.New()
	if _, err := cache.Parse(path, src); err != nil {
		t.Fatalf("parse source: %v", err)
	}

	ctx := &pass.Context{
		ASTCache:     cache,
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
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	src := []byte("package foo\nfunc f() {\n\tx := 1\n\t_ = x\n}\n")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	cache := astcache.New()
	if _, err := cache.Parse(path, src); err != nil {
		t.Fatalf("parse source: %v", err)
	}
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
			fc := pass.FileChange{Path: "foo.go", Extension: ".go", IsTest: testCase.test}
			got := th.checkFile(cache.FileSet(), fc, file)
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
	test   bool
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
		{
			name:   "long func in a production file",
			source: "package foo\nfunc tbl() {\n" + strings.Repeat("\tprintln()\n", 98) + "}\n",
			expect: 1,
		},
		{
			name:   "test file relaxes only the length",
			source: "package foo\nfunc tbl() {\n" + strings.Repeat("\tprintln()\n", 98) + "}\n",
			test:   true,
			expect: 0,
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

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{
			name:   "no branches",
			source: "package foo\nfunc f() { _ = 1 }\n",
			want:   1,
		},
		{
			name:   "one if",
			source: "package foo\nfunc f() { if true {} }\n",
			want:   2,
		},
		{
			name:   "for loop",
			source: "package foo\nfunc f() { for i := 0; i < 10; i++ {} }\n",
			want:   2,
		},
		{
			name:   "range loop",
			source: "package foo\nfunc f() { for range []int{} {} }\n",
			want:   2,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, file := parseSourceFile(t, testCase.source)
			fn, ok := file.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("Decls[0] = %T, want *ast.FuncDecl", file.Decls[0])
			}
			got := cyclomaticComplexity(fn)
			if got != testCase.want {
				t.Fatalf("got complexity %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestCountParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{"no params", "package foo\nfunc f() {}\n", 0},
		{"one param", "package foo\nfunc f(x int) {}\n", 1},
		{"two params", "package foo\nfunc f(x, y int) {}\n", 2},
		{"two groups", "package foo\nfunc f(x int, y string) {}\n", 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, file := parseSourceFile(t, testCase.source)
			fn, ok := file.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("Decls[0] = %T, want *ast.FuncDecl", file.Decls[0])
			}
			got := countParams(fn.Type.Params)
			if got != testCase.want {
				t.Fatalf("got %d params, want %d", got, testCase.want)
			}
		})
	}
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
