package gosyntax

import (
	"go/ast"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestFunctions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want []syntax.Function
	}{
		{
			name: "plain function",
			body: "package x\n\nfunc run(a int, b, c string) {\n\t_ = a\n}\n",
			want: []syntax.Function{{
				Name: "run", Kind: syntax.FuncFunction, Line: 3, BodyStart: 3, BodyEnd: 5,
				Params: 3, Complexity: 1,
			}},
		},
		{
			name: "method with unnamed params",
			body: "package x\n\ntype T struct{}\n\nfunc (T) run(int, string) {}\n",
			want: []syntax.Function{{
				Name: "run", Kind: syntax.FuncMethod, Line: 5, BodyStart: 5, BodyEnd: 5,
				Params: 2, Complexity: 1,
			}},
		},
		{
			name: "branches including a nested literal",
			body: "package x\n\nfunc run(ok bool, rows []int) {\n" +
				"\tif ok && len(rows) > 0 {\n\t\tfor range rows {\n\t\t}\n\t}\n" +
				"\tfn := func() {\n\t\tif ok || false {\n\t\t}\n\t}\n\t_ = fn\n" +
				"\tswitch {\n\tcase ok:\n\tdefault:\n\t}\n" +
				"\tselect {\n\tcase <-make(chan int):\n\t}\n}\n",
			want: []syntax.Function{{
				Name: "run", Kind: syntax.FuncFunction, Line: 3, BodyStart: 3, BodyEnd: 20,
				Params: 2, Complexity: 8,
			}},
		},
		{name: "bodiless declaration is skipped", body: "package x\n\nfunc run()\n"},
		{name: "no functions", body: "package x\n\nvar v int\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, plainSource(), "").Functions
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("Functions = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func firstFunc(t *testing.T, source string) *ast.FuncDecl {
	t.Helper()

	_, file := parseSource(t, source)
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Decls[0] = %T, want *ast.FuncDecl", file.Decls[0])
	}
	return fn
}
