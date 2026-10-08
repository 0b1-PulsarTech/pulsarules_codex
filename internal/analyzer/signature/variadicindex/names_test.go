package variadicindex

import (
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
