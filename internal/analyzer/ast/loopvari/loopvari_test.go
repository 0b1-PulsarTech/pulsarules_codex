package loopvari

import (
	"fmt"
	"go/ast"
	"go/parser"
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

	a := NewAnalyzer()
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
			if got := a.Analyze(testCase.ctx); len(got) != 0 {
				t.Fatalf("got %d findings, want 0", len(got))
			}
		})
	}
}

type loopCase struct {
	name   string
	body   string
	decls  string
	expect int
}

func runLoopCases(t *testing.T, testCases []loopCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			source := "package foo\nimport \"fmt\"\nvar _ = fmt.Sprint\n" +
				"func f(s []int, m map[int]int, n int) (out []int) {\n" + testCase.body +
				"\n\treturn\n}\n" + testCase.decls + "\n"
			got := analyzeSource(t, source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(indexingCases(), escapeCases()...)
	testCases = append(testCases, postCases()...)
	runLoopCases(t, testCases)
}

func indexingCases() []loopCase {
	return []loopCase{
		{name: "range index only", body: "for i := range s {\n s[i]++\n}"},
		{name: "range index with offset", body: "for i := range s {\n _ = s[i+1] + s[i-1]\n}"},
		{name: "range index in parens", body: "for i := range s {\n _ = s[(i+1)]\n}"},
		{name: "range index on map", body: "for i := range s {\n m[i] = m[i] + 1\n}"},
		{name: "range slice bounds", body: "for i := range s {\n _ = s[i:]\n}"},
		{
			name: "range compared to len",
			body: "for i := range s {\n if i+1 < len(s) {\n  _ = s[i+1]\n }\n}",
		},
		{
			name: "range compared to len minus one",
			body: "for i := range s {\n if i == len(s)-1 {\n  _ = s[i]\n }\n}",
		},
		{name: "for index with bound cond", body: "for i := 0; i < len(s); i++ {\n s[i]++\n}"},
		{name: "for with plain bound cond", body: "for i := 0; i < n; i++ {\n s[i]++\n}"},
		{name: "for with step assign", body: "for i := 0; i < n; i += 2 {\n s[i]++\n}"},
		{name: "for decrement", body: "for i := n - 1; i >= 0; i-- {\n s[i]++\n}"},
		{
			name: "other loop variable name",
			body: "for row := range s {\n out = append(out, row)\n}",
		},
		{name: "range with value only", body: "for _, i := range s {\n out = append(out, i)\n}"},
		{
			name: "assign to range key without define",
			body: "var i int\nfor i = range s {\n out = append(out, i)\n}",
		},
	}
}

func escapeCases() []loopCase {
	return []loopCase{
		{
			name:   "call argument",
			body:   "for i := range s {\n _ = fmt.Sprintf(\"%d\", i)\n}",
			expect: 1,
		},
		{name: "append value", body: "for i := range s {\n out = append(out, i)\n}", expect: 1},
		{
			name:   "assigned to a variable",
			body:   "for i := range s {\n x := i\n _ = x\n}",
			expect: 1,
		},
		{name: "assigned to existing variable", body: "for i := range s {\n n = i\n}", expect: 1},
		{name: "stored as map value", body: "for i := range s {\n m[0] = i\n}", expect: 1},
		{
			name:   "returned",
			body:   "for i := range s {\n if s[i] > 0 {\n  return []int{i}\n }\n}",
			expect: 1,
		},
		{
			name:   "compared with zero",
			body:   "for i := range s {\n if i == 0 {\n  continue\n }\n}",
			expect: 1,
		},
		{
			name:   "compared greater than zero",
			body:   "for i := range s {\n if i > 0 {\n  _ = s[i-1]\n }\n}",
			expect: 1,
		},
		{name: "modulo", body: "for i := range s {\n if i%2 == 0 {\n  continue\n }\n}", expect: 1},
		{
			name:   "compared with a plain variable in body",
			body:   "for i := range s {\n if i < n {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
		{
			name:   "arithmetic passed to a call",
			body:   "for i := range s {\n _ = fmt.Sprint(i + 1)\n}",
			expect: 1,
		},
		{
			name:   "captured by a closure",
			body:   "for i := range s {\n go func() {\n  _ = s[i]\n }()\n}",
			expect: 1,
		},
		{
			name:   "for loop call argument",
			body:   "for i := 0; i < n; i++ {\n _ = fmt.Sprint(i)\n}",
			expect: 1,
		},
		{
			name:   "for loop assigned",
			body:   "for i := 0; i < n; i++ {\n out = append(out, i)\n}",
			expect: 1,
		},
		{
			name:   "for loop post uses i in a call",
			body:   "for i := 0; i < n; n = len(fmt.Sprint(i)) {\n}",
			expect: 1,
		},
		{
			name:   "index on the left and i on the right",
			body:   "for i := range s {\n s[i] = i\n}",
			expect: 1,
		},
		{
			name:   "one finding per loop",
			body:   "for i := range s {\n _ = fmt.Sprint(i, i)\n}",
			expect: 1,
		},
		{
			name:   "i used as an operand of a call receiver",
			body:   "for i := range s {\n _ = fmt.Sprint(s[i], i)\n}",
			expect: 1,
		},
	}
}

func postCases() []loopCase {
	return []loopCase{
		{
			name: "post assigns a call of i",
			body: "g := func(x int) int { return x }\nfor i := 0; i < n; i = g(i) {\n" +
				" _ = s[i]\n}",
			expect: 1,
		},
		{name: "post offset assignment", body: "for i := 0; i < n; i = i + 1 {\n _ = s[i]\n}"},
		{name: "post multiply assign", body: "for i := 1; i < n; i *= 2 {\n _ = s[i]\n}"},
		{
			name:   "post op assign mentioning i",
			body:   "for i := 1; i < n; i += i {\n _ = s[i]\n}",
			expect: 1,
		},
		{
			name: "two pointer post",
			body: "for i, j := 0, n-1; i < j; i, j = i+1, j-1 {\n _ = s[i] + s[j]\n}",
		},
		{
			name:   "multi assign post swapping i",
			body:   "for i, j := 0, 1; i < n; i, j = j, i+j {\n _ = s[i]\n}",
			expect: 1,
		},
		{name: "post steps another variable", body: "for i := 0; i < n; n-- {\n _ = s[i]\n}"},
		{name: "range over an int", body: "for i := range 10 {\n _ = s[i]\n}"},
		{
			name:   "range over an int used in a call",
			body:   "for i := range 10 {\n _ = fmt.Sprint(i)\n}",
			expect: 1,
		},
		{
			name: "unused key with the value used",
			body: "for i, v := range s {\n _ = s[i]\n out = append(out, v)\n}",
		},
		{
			name:   "key escapes next to the value",
			body:   "for i, v := range s {\n out = append(out, v+i)\n}",
			expect: 1,
		},
		{name: "defer argument", body: "for i := range s {\n defer fmt.Println(i)\n}", expect: 1},
		{name: "go argument", body: "for i := range s {\n go fmt.Println(i)\n}", expect: 1},
		{
			name: "post adds i on the right",
			body: "for i := 0; i < n; i = 1 + i {\n _ = s[i]\n}",
		},
		{
			name:   "post subtracts i from a constant",
			body:   "for i := 0; i < n; i = 1 - i {\n _ = s[i]\n}",
			expect: 1,
		},
	}
}

func parseDecls(t *testing.T, source string) *ast.File {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), "foo.go", source, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f
}

func funcBody(t *testing.T, f *ast.File, index int) *ast.BlockStmt {
	t.Helper()

	fn, ok := f.Decls[index].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("declaration %d is not a function", index)
	}
	return fn.Body
}

func parseFuncLit(t *testing.T, source string) *ast.FuncLit {
	t.Helper()

	expr, err := parser.ParseExpr(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lit, ok := expr.(*ast.FuncLit)
	if !ok {
		t.Fatalf("%q is not a function literal", source)
	}
	return lit
}

func analyzeSource(t *testing.T, source string) []pass.Finding {
	t.Helper()

	cache := astcache.New()
	f, err := cache.Parse("foo.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	typeCheck(t, cache.FileSet(), f)
	reporter := pass.NewReporter("loop-var-i", pass.SeverityWarning, pass.CatAST)
	return checkFile(
		cache.FileSet(),
		pass.FileChange{Path: "foo.go", Extension: ".go"},
		f,
		reporter,
	)
}

type fmtStub struct{}

func (fmtStub) Import(path string) (*types.Package, error) {
	if path != "fmt" {
		return nil, fmt.Errorf("unexpected import %q", path)
	}
	pkg := types.NewPackage("fmt", "fmt")
	args := types.NewVar(token.NoPos, pkg, "a", types.NewSlice(types.NewInterfaceType(nil, nil)))
	result := types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])
	sig := types.NewSignatureType(
		nil, nil, nil, types.NewTuple(args), types.NewTuple(result), true,
	)
	for _, name := range []string{"Sprint", "Sprintf", "Println"} {
		pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, name, sig))
	}
	pkg.MarkComplete()
	return pkg, nil
}

func typeCheck(t *testing.T, fset *token.FileSet, f *ast.File) {
	t.Helper()

	config := types.Config{Importer: fmtStub{}}
	if _, err := config.Check("foo", fset, []*ast.File{f}, nil); err != nil {
		t.Fatalf("test input does not compile: %v", err)
	}
}

func TestAnalyze_EndToEnd(t *testing.T) {
	t.Parallel()

	sourceItem := "package foo\nimport \"fmt\"\nfunc f(s []int) {\n\tfor i := range s {\n" +
		"\t\t_ = fmt.Sprint(i)\n\t}\n}\n"
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
