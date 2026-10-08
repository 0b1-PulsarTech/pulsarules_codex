package goast

import "go/ast"

func FieldsDeclare(list *ast.FieldList, name string) bool {
	if list == nil {
		return false
	}
	for _, field := range list.List {
		if IdentsNamed(field.Names, name) {
			return true
		}
	}
	return false
}

func IdentsNamed(idents []*ast.Ident, name string) bool {
	for _, ident := range idents {
		if ident.Name == name {
			return true
		}
	}
	return false
}

func ExprsMention(exprs []ast.Expr, name string) bool {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

func Any(root ast.Node, match func(ast.Node) bool) bool {
	return AnyUntil(root, match, func(ast.Node) bool { return false })
}

func AnyUntil(root ast.Node, match, boundary func(ast.Node) bool) bool {
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
		if found || node == nil {
			return false
		}
		found = match(node)
		return !found && !boundary(node)
	})
	return found
}
