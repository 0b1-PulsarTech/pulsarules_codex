package complexity

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *core.AnalysisContext
		expect int
	}{
		{
			name: "no cache",
			ctx:  &core.AnalysisContext{},
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ASTCache:     astcache.New(),
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := a.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

// TestAnalyze_HonoursConfiguredThresholds asserts a tightened max_func_lines
// config param, not just the compiled-in default, decides whether a function
// is reported.
func TestAnalyze_HonoursConfiguredThresholds(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
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

	ctx := &core.AnalysisContext{
		ASTCache:     cache,
		ChangedFiles: []core.FileChange{{Path: path, Extension: ".go"}},
	}
	if got := len(a.Analyze(ctx)); got != 0 {
		t.Fatalf("expected 0 findings against the default threshold, got %d", got)
	}

	ctx.Config = &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
		"complexity": {Params: map[string]any{"max_func_lines": 1}},
	}}
	got := a.Analyze(ctx)
	if !hasFuncLinesFinding(got) {
		t.Fatalf("a configured max_func_lines=1 should report the 4-line func, got %+v", got)
	}
}

func hasFuncLinesFinding(findings []core.Finding) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Message, "is 4 lines, max 1") {
			return true
		}
	}
	return false
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	th := thresholds{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
	}

	for _, testCase := range checkFileTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cache, f := parseSourceFile(t, testCase.source)
			fc := core.FileChange{Path: "foo.go", Extension: ".go"}
			got := th.checkFile(cache.FileSet(), fc, f)
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

// checkFileCasesThresholds covers the three plain threshold checks
// (complexity, func length via magic number, param count) plus the
// complexity-high case.
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

// checkFileCasesFlagArgs covers the value-arg-name exclusion heuristic in
// checkFlagArguments. The magic-number heuristic cases live alongside
// findMagicNumbers in magicnumber_test.go.
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
			_, f := parseSourceFile(t, testCase.source)
			fn, ok := f.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("Decls[0] = %T, want *ast.FuncDecl", f.Decls[0])
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
			_, f := parseSourceFile(t, testCase.source)
			fn, ok := f.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("Decls[0] = %T, want *ast.FuncDecl", f.Decls[0])
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
	f, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}
	return cache, f
}
