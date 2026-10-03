package toomanyresults

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	testCases := []struct {
		name string
		ctx  *core.AnalysisContext
	}{
		{name: "no cache and no files", ctx: &core.AnalysisContext{}},
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
			if got := analyzer.Analyze(testCase.ctx); len(got) != 0 {
				t.Fatalf("got %d findings, want 0", len(got))
			}
		})
	}
}

type checkCase struct {
	name   string
	source string
	limit  int
	expect int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(firstCases(), secondCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeSource(t, testCase.source, testCase.limit)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func firstCases() []checkCase {
	return []checkCase{
		{
			name:   "three results ok",
			source: "package foo\nfunc f() (int, string, error) { return 0, \"\", nil }\n",
			limit:  3,
		},
		{
			name: "four results flagged",
			source: "package foo\nfunc f() (int, string, bool, error) " +
				"{ return 0, \"\", false, nil }\n",
			limit:  3,
			expect: 1,
		},
		{
			name:   "one field with four names flagged",
			source: "package foo\nfunc f() (a, b, c, d int) { return }\n",
			limit:  3,
			expect: 1,
		},
		{
			name:   "grouped names count individually",
			source: "package foo\nfunc f() (a, b int, c string) { return }\n",
			limit:  3,
		},
		{
			name: "method flagged",
			source: "package foo\ntype T struct{}\nfunc (T) f() (int, int, int, int) " +
				"{ return 0, 0, 0, 0 }\n",
			limit:  3,
			expect: 1,
		},
		{
			name:   "interface method flagged",
			source: "package foo\ntype T interface {\n\tF() (int, int, int, int)\n}\n",
			limit:  3,
			expect: 1,
		},
	}
}

func secondCases() []checkCase {
	return []checkCase{
		{
			name:   "interface method within the limit",
			source: "package foo\ntype T interface {\n\tF() (int, error)\n}\n",
			limit:  3,
		},
		{
			name:   "embedded interface ignored",
			source: "package foo\ntype R interface{ Read() int }\ntype T interface {\n\tR\n}\n",
			limit:  3,
		},
		{
			name:   "func literal flagged",
			source: "package foo\nvar f = func() (int, int, int, int) { return 0, 0, 0, 0 }\n",
			limit:  3,
			expect: 1,
		},
		{
			name:   "no results",
			source: "package foo\nfunc f() {}\n",
			limit:  3,
		},
		{
			name:   "func type declaration flagged",
			source: "package foo\ntype H func() (int, int, int, int)\n",
			limit:  3,
			expect: 1,
		},
		{
			name:   "func type declaration within the limit",
			source: "package foo\ntype H func() (int, int)\n",
			limit:  3,
		},
		{
			name:   "struct field func type out of scope",
			source: "package foo\ntype S struct{ F func() (int, int, int, int) }\n",
			limit:  3,
		},
		{
			name:   "func typed parameter out of scope",
			source: "package foo\nfunc f(g func() (int, int, int, int)) {}\n",
			limit:  3,
		},
		{
			name:   "non func type declaration",
			source: "package foo\ntype H int\n",
			limit:  3,
		},
	}
}

func analyzeSource(t *testing.T, source string, limit int) []core.Finding {
	t.Helper()

	cache := astcache.New()
	file, err := cache.Parse("foo.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	typeCheck(t, cache.FileSet(), file)
	reporter := core.NewReporter("too-many-results", core.SeverityWarning, core.CatAST)
	return rule{reporter: reporter, limit: limit}.check(
		cache.FileSet(),
		core.FileChange{Path: "foo.go", Extension: ".go"},
		file,
	)
}

func buildCtx(t *testing.T, source string, params core.ParamSet) *core.AnalysisContext {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("foo.go", []byte(source)); err != nil {
		t.Fatal(err)
	}
	ctx := &core.AnalysisContext{
		ASTCache: cache,
		Sources: sources.NewFSSourceProvider(fstest.MapFS{
			"foo.go": &fstest.MapFile{Data: []byte(source)},
		}, "", sources.SkipSet{}),
		ChangedFiles: []core.FileChange{{Path: "foo.go", Extension: ".go"}},
	}
	if params != nil {
		ctx.Config = &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
			"too-many-results": {Params: params},
		}}
	}
	return ctx
}

func typeCheck(t *testing.T, fset *token.FileSet, file *ast.File) {
	t.Helper()

	config := types.Config{}
	if _, err := config.Check("foo", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("test input does not compile: %v", err)
	}
}

func TestAnalyze_EndToEnd(t *testing.T) {
	t.Parallel()

	two := "package foo\nfunc f() (int, error) { return 0, nil }\n"
	four := "package foo\nfunc f() (int, int, int, int) { return 0, 0, 0, 0 }\n"
	testCases := []struct {
		name      string
		source    string
		generated bool
		params    core.ParamSet
		expect    int
	}{
		{name: "default limit flags four", source: four, expect: 1},
		{name: "default limit passes two", source: two},
		{name: "generated file skipped", source: four, generated: true},
		{
			name:   "max_results lowers the limit",
			source: two, params: core.ParamSet{"max_results": 1}, expect: 1,
		},
		{
			name:   "max_results raises the limit",
			source: four, params: core.ParamSet{"max_results": 4},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := buildCtx(t, testCase.source, testCase.params)
			ctx.ChangedFiles[0].Generated = testCase.generated
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}
