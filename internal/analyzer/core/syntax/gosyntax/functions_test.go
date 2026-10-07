package gosyntax

import (
	"go/ast"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{name: "no branches", source: "package foo\nfunc f() { _ = 1 }\n", want: 1},
		{name: "one if", source: "package foo\nfunc f() { if true {} }\n", want: 2},
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

			got := cyclomaticComplexity(firstFunc(t, testCase.source))
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
		{name: "no params", source: "package foo\nfunc f() {}\n"},
		{name: "one param", source: "package foo\nfunc f(x int) {}\n", want: 1},
		{name: "two params", source: "package foo\nfunc f(x, y int) {}\n", want: 2},
		{name: "two groups", source: "package foo\nfunc f(x int, y string) {}\n", want: 2},
		{name: "unnamed", source: "package foo\nfunc f(int, string) {}\n", want: 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := countParams(firstFunc(t, testCase.source).Type.Params)
			if got != testCase.want {
				t.Fatalf("got %d params, want %d", got, testCase.want)
			}
		})
	}
}
