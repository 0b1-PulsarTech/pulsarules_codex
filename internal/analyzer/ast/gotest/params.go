package gotest

import "go/ast"

func matchesSignature(prefix string, ft *ast.FuncType) bool {
	want := map[string]string{
		"Test":      "testing.T",
		"Benchmark": "testing.B",
		"Fuzz":      "testing.F",
	}
	wantType, ok := want[prefix]
	if !ok {
		return ft.Params == nil || len(ft.Params.List) == 0
	}
	if ft.Params == nil || len(ft.Params.List) != 1 {
		return false
	}
	return paramTypeName(ft.Params.List[0].Type) == wantType
}

func paramTypeName(expr ast.Expr) string {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return ""
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name + "." + sel.Sel.Name
}
