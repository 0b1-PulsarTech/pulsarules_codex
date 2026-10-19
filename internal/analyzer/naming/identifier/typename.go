package identifier

import (
	"go/ast"
	"strings"
)

func typeOfExpr(expr ast.Expr) string {
	// No go/types: loading packages costs more than this rule is worth.
	switch typed := expr.(type) {
	case *ast.CompositeLit:
		return typeName(typed.Type)
	case *ast.CallExpr:
		return strings.TrimPrefix(typeName(typed.Fun), "New")
	case *ast.UnaryExpr:
		return typeOfExpr(typed.X)
	}
	return ""
}

func typeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return typeName(typed.X)
	case *ast.SelectorExpr:
		if typed.Sel == nil {
			return ""
		}
		return typed.Sel.Name
	case *ast.IndexExpr:
		return typeName(typed.X)
	case *ast.IndexListExpr:
		return typeName(typed.X)
	case *ast.ArrayType:
		return typeName(typed.Elt)
	}
	return ""
}
