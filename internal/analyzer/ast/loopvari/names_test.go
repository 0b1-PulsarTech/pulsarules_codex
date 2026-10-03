package loopvari

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestIsCounterValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		tok  token.Token
		rhs  string
		want bool
	}{
		{name: "op assign without i", tok: token.MUL_ASSIGN, rhs: "2", want: true},
		{name: "op assign with i", tok: token.ADD_ASSIGN, rhs: "i"},
		{name: "step forward", tok: token.ASSIGN, rhs: "i + 1", want: true},
		{name: "step from the left constant", tok: token.ASSIGN, rhs: "1 + i", want: true},
		{name: "step back", tok: token.ASSIGN, rhs: "i - 1", want: true},
		{name: "constant minus i", tok: token.ASSIGN, rhs: "1 - i"},
		{name: "step by i", tok: token.ASSIGN, rhs: "i + i"},
		{name: "multiply", tok: token.ASSIGN, rhs: "i * 2"},
		{name: "call", tok: token.ASSIGN, rhs: "g(i)"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			expr, err := parser.ParseExpr(testCase.rhs)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := isCounterValue(testCase.tok, expr); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestLoopVarDecl(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "declares i", source: "var i int", want: true},
		{name: "declares i in a group", source: "var (\n a = 1\n i = 2\n)", want: true},
		{name: "declares another name", source: "var j int"},
		{name: "constant group", source: "const j = 1"},
		{name: "type declaration", source: "type i int"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f := parseDecls(t, "package foo\nfunc f() {\n"+testCase.source+"\n}")
			body := funcBody(t, f, 0)
			if got := loopVarDecl(body.List[0]) != nil; got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestLoopVarDecl_NotADeclaration(t *testing.T) {
	t.Parallel()

	f := parseDecls(t, "package foo\nfunc f() {\ni := 1\n_ = i\n}")
	body := funcBody(t, f, 0)
	if loopVarDecl(body.List[0]) != nil {
		t.Fatal("a define statement is not a var declaration")
	}
}

func TestFieldsNameLoopVar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{name: "parameter i", source: "func(i int) {}", want: true},
		{name: "parameter in a group", source: "func(a, i int) {}", want: true},
		{name: "other parameter", source: "func(a int) {}"},
		{name: "no parameters", source: "func() {}"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			lit := parseFuncLit(t, testCase.source)
			if got := fieldsNameLoopVar(lit.Type.Params); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFieldsNameLoopVar_Nil(t *testing.T) {
	t.Parallel()

	if fieldsNameLoopVar(nil) {
		t.Fatal("a nil field list names nothing")
	}
}
