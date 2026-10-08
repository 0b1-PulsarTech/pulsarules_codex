package goast

import (
	"go/ast"
	"testing"
)

func TestFieldsDeclare(t *testing.T) {
	t.Parallel()

	file := parse(t, "func f(first, second int, third string) {}")
	params := firstFunc(t, file).Type.Params
	testCases := []struct {
		name   string
		fields *ast.FieldList
		ident  string
		want   bool
	}{
		{name: "declared", fields: params, ident: "second", want: true},
		{name: "absent", fields: params, ident: "other"},
		{name: "nil list", ident: "first"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := FieldsDeclare(testCase.fields, testCase.ident); got != testCase.want {
				t.Fatalf("FieldsDeclare = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIdentsNamed(t *testing.T) {
	t.Parallel()

	file := parse(t, "func f(first, second int) {}\nfunc g(int) {}")
	named := firstFunc(t, file).Type.Params.List
	unnamed, ok := file.Decls[1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl 1 = %T, want *ast.FuncDecl", file.Decls[1])
	}
	testCases := []struct {
		name   string
		idents []*ast.Ident
		ident  string
		want   bool
	}{
		{name: "declared", idents: named[0].Names, ident: "second", want: true},
		{name: "absent", idents: named[0].Names, ident: "other"},
		{
			name:   "unnamed parameter has no idents",
			idents: unnamed.Type.Params.List[0].Names,
			ident:  "int",
		},
		{name: "nil slice", ident: "first"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IdentsNamed(testCase.idents, testCase.ident); got != testCase.want {
				t.Fatalf("IdentsNamed = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestExprsMention(t *testing.T) {
	t.Parallel()

	file := parse(t, "func f() { first, other.second, third := 1, 2, 3 }")
	assign, ok := firstFunc(t, file).Body.List[0].(*ast.AssignStmt)
	if !ok {
		t.Fatal("first statement is not an assignment")
	}
	testCases := []struct {
		name  string
		ident string
		want  bool
	}{
		{name: "bare identifier", ident: "third", want: true},
		{name: "selector is not an identifier", ident: "second"},
		{name: "absent", ident: "missing"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := ExprsMention(assign.Lhs, testCase.ident); got != testCase.want {
				t.Fatalf("ExprsMention = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAny(t *testing.T) {
	t.Parallel()

	file := parse(t, "func f() { a := 1; _ = a }")
	testCases := []struct {
		name  string
		match func(ast.Node) bool
		want  bool
	}{
		{
			name:  "found",
			match: func(node ast.Node) bool { _, ok := node.(*ast.AssignStmt); return ok },
			want:  true,
		},
		{
			name:  "absent",
			match: func(node ast.Node) bool { _, ok := node.(*ast.GoStmt); return ok },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Any(file, testCase.match); got != testCase.want {
				t.Fatalf("Any = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAnyUntil(t *testing.T) {
	t.Parallel()

	file := parse(t, "func f() { go func() { a := 1; _ = a }() }")
	isAssign := func(node ast.Node) bool { _, ok := node.(*ast.AssignStmt); return ok }
	isLit := func(node ast.Node) bool { _, ok := node.(*ast.FuncLit); return ok }
	testCases := []struct {
		name     string
		match    func(ast.Node) bool
		boundary func(ast.Node) bool
		want     bool
	}{
		{
			name:     "walks into a non-boundary",
			match:    isAssign,
			boundary: func(ast.Node) bool { return false },
			want:     true,
		},
		{name: "does not walk into the boundary", match: isAssign, boundary: isLit},
		{name: "the boundary itself is matched", match: isLit, boundary: isLit, want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := AnyUntil(file, testCase.match, testCase.boundary); got != testCase.want {
				t.Fatalf("AnyUntil = %v, want %v", got, testCase.want)
			}
		})
	}
}

func firstFunc(t *testing.T, file *ast.File) *ast.FuncDecl {
	t.Helper()

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl 0 = %T, want *ast.FuncDecl", file.Decls[0])
	}
	return fn
}
