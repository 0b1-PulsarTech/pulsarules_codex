package interfacegroup

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
	name     string
	source   string
	expect   int
	wantLine int
	wantMsg  string
}

func checkCases() []checkCase {
	return []checkCase{
		{
			name:     "two ungrouped flagged at the first",
			source:   "package foo\ntype A interface{ F() }\ntype B interface{ G() }\n",
			expect:   1,
			wantLine: 2,
			wantMsg:  "file declares 2 interfaces outside a single type (...) block",
		},
		{
			name:   "grouped in one block is fine",
			source: "package foo\ntype (\n\tA interface{ F() }\n\tB interface{ G() }\n)\n",
		},
		{
			name: "grouped block plus a stray flagged at the stray",
			source: "package foo\ntype (\n\tA interface{ F() }\n\tB interface{ G() }\n)\n" +
				"type C interface{ H() }\n",
			expect:   1,
			wantLine: 6,
			wantMsg:  "file declares 1 interface outside a single type (...) block",
		},
		{
			name:     "block of one plus a stray flagged once",
			source:   "package foo\ntype (\n\tA interface{ F() }\n)\ntype B interface{ G() }\n",
			expect:   1,
			wantLine: 5,
			wantMsg:  "file declares 1 interface outside a single type (...) block",
		},
		{
			name:   "alias to an interface literal is not a declaration",
			source: "package foo\ntype A = interface{ F() }\ntype B interface{ G() }\n",
		},
		{
			name: "alias does not count toward the total",
			source: "package foo\ntype A = interface{ F() }\ntype B interface{ G() }\n" +
				"type C interface{ H() }\n",
			expect:   1,
			wantLine: 3,
			wantMsg:  "file declares 2 interfaces outside a single type (...) block",
		},
		{
			name:   "alias inside the group is not a declaration",
			source: "package foo\ntype (\n\tA = interface{ F() }\n\tB interface{ G() }\n)\n",
		},
		{
			name: "two stray blocks of one",
			source: "package foo\ntype (\n\tA interface{ F() }\n)\ntype (\n" +
				"\tB interface{ G() }\n)\n",
			expect:   1,
			wantLine: 6,
			wantMsg:  "file declares 1 interface outside a single type (...) block",
		},
		{
			name:   "single interface is fine",
			source: "package foo\ntype A interface{ F() }\n",
		},
		{
			name:   "non-interface types ignored",
			source: "package foo\ntype A struct{}\ntype B int\ntype C interface{ F() }\n",
		},
		{
			name: "interfaces mixed with structs in one block",
			source: "package foo\ntype (\n\tA interface{ F() }\n\tS struct{}\n" +
				"\tB interface{ G() }\n)\n",
		},
		{
			name:   "no type declarations",
			source: "package foo\nvar x = 1\nfunc f() {}\n",
		},
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := checkCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeSource(t, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
			if testCase.expect > 0 && got[0].Line != testCase.wantLine {
				t.Fatalf("line = %d, want %d", got[0].Line, testCase.wantLine)
			}
			if testCase.expect > 0 && got[0].Message != testCase.wantMsg {
				t.Fatalf("message = %q, want %q", got[0].Message, testCase.wantMsg)
			}
		})
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
	reporter := pass.NewReporter("ungrouped-interfaces", pass.SeverityWarning, pass.CatAST)
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

	sourceItem := "package foo\ntype A interface{ F() }\ntype B interface{ G() }\n"
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
