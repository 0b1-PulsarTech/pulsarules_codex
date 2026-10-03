package variadicindex

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	testCases := []struct {
		name string
		ctx  *pass.Context
	}{
		{name: "no cache and no files", ctx: &pass.Context{}},
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
			if got := analyzer.Analyze(testCase.ctx); len(got) != 0 {
				t.Fatalf("got %d findings, want 0", len(got))
			}
		})
	}
}

type checkCase struct {
	name   string
	source string
	expect int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := flaggedCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeSource(t, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestSuggestion(t *testing.T) {
	t.Parallel()

	const want = "take a named options struct, or range over the values instead of indexing them"
	testCases := []checkCase{
		{
			name:   "plain index",
			source: "package foo\nfunc f(opts ...int) int { return opts[0] }\n",
		},
		{
			name: "index inside a range over the same variadic",
			source: "package foo\nfunc f(opts ...int) (n int) {\n\tfor i := range opts {\n" +
				"\t\tn += opts[i]\n\t}\n\treturn\n}\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeSource(t, testCase.source)
			if len(got) != 1 || got[0].Suggestion != want {
				t.Fatalf("got %v, want one finding suggesting %q", got, want)
			}
		})
	}
}

func flaggedCases() []checkCase {
	return []checkCase{
		{
			name:   "first element flagged",
			source: "package foo\nfunc f(opts ...int) int { return opts[0] }\n",
			expect: 1,
		},
		{
			name:   "any index flagged",
			source: "package foo\nfunc f(n int, opts ...int) int { return opts[n] + opts[1] }\n",
			expect: 2,
		},
		{
			name: "method flagged",
			source: "package foo\ntype T struct{}\nfunc (T) f(opts ...int) int { return " +
				"opts[0] }\n",
			expect: 1,
		},
		{
			name:   "func literal flagged",
			source: "package foo\nvar f = func(opts ...int) int { return opts[0] }\n",
			expect: 1,
		},
		{
			name: "index inside nested block flagged",
			source: "package foo\nfunc f(opts ...int) int {\n\tif len(opts) > 0 {\n" +
				"\t\treturn opts[0]\n\t}\n\treturn 0\n}\n",
			expect: 1,
		},
		{
			name:   "len is fine",
			source: "package foo\nfunc f(opts ...int) int { return len(opts) }\n",
		},
		{
			name: "range is fine",
			source: "package foo\nfunc f(opts ...int) (n int) {\n" +
				"\tfor _, o := range opts {\n\t\tn += o\n\t}\n\treturn\n}\n",
		},
		{
			name:   "slicing is not flagged",
			source: "package foo\nfunc f(opts ...int) []int { return opts[1:] }\n",
		},
		{
			name:   "non-variadic last param ignored",
			source: "package foo\nfunc f(opts []int) int { return opts[0] }\n",
		},
	}
}

func analyzeSource(t *testing.T, source string) []pass.Finding {
	t.Helper()

	cache := astcache.New()
	file, err := cache.Parse("foo.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	typeCheck(t, cache.FileSet(), file)
	reporter := pass.NewReporter("variadic-index", pass.SeverityWarning, pass.CatAST)
	return checkFile(
		cache.FileSet(),
		pass.FileChange{Path: "foo.go", Extension: ".go"},
		file,
		reporter,
	)
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

	sourceItem := "package foo\nfunc f(opts ...int) int { return opts[0] }\n"
	testCases := []struct {
		name      string
		generated bool
		expect    int
	}{
		{name: "hand written file flagged", expect: 1},
		{name: "generated file skipped", generated: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cache := astcache.New()
			if _, err := cache.Parse("foo.go", []byte(sourceItem)); err != nil {
				t.Fatal(err)
			}
			ctx := &pass.Context{
				ASTCache: cache,
				Sources: source.NewFSProvider(fstest.MapFS{
					"foo.go": &fstest.MapFile{Data: []byte(sourceItem)},
				}, "", source.SkipSet{}),
				ChangedFiles: []pass.FileChange{
					{Path: "foo.go", Extension: ".go", Generated: testCase.generated},
				},
			}
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}
