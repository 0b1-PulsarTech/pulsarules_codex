package variadicindex

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestScopeIsTarget(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		expr string
		want bool
	}{
		{name: "plain name", expr: "opts", want: true},
		{name: "parenthesised name", expr: "((opts))", want: true},
		{name: "other name", expr: "other"},
		{name: "selector", expr: "x.opts"},
		{name: "call", expr: "opts()"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			expr, err := parser.ParseExpr(testCase.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := (scope{name: "opts"}).isTarget(expr); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestScopeHasField(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "named parameter", source: "func(opts int) {}", want: true},
		{name: "grouped parameter", source: "func(a, opts int) {}", want: true},
		{name: "other parameter", source: "func(a int) {}"},
		{name: "unnamed parameter", source: "func(int) {}"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			lit := parseFuncLit(t, testCase.source)
			if got := (scope{name: "opts"}).hasField(lit.Type.Params); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestScopeHasField_Nil(t *testing.T) {
	t.Parallel()

	if (scope{name: "opts"}).hasField(nil) {
		t.Fatal("a nil field list has no name")
	}
}

func TestScopeHasExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		exprs []ast.Expr
		want  bool
	}{
		{
			name:  "matching identifier",
			exprs: []ast.Expr{ast.NewIdent("_"), ast.NewIdent("opts")},
			want:  true,
		},
		{name: "other identifier", exprs: []ast.Expr{ast.NewIdent("other")}},
		{name: "non identifier", exprs: []ast.Expr{&ast.BasicLit{Value: "1"}}},
		{name: "empty list"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := (scope{name: "opts"}).hasExpr(testCase.exprs); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestScopeHasName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		idents []*ast.Ident
		want   bool
	}{
		{name: "matching name", idents: []*ast.Ident{ast.NewIdent("opts")}, want: true},
		{name: "other name", idents: []*ast.Ident{ast.NewIdent("other")}},
		{name: "empty list"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := (scope{name: "opts"}).hasName(testCase.idents); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
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
