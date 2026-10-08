package goast

import (
	"go/ast"
	"slices"
)

func PkgCall(file *ast.File, call *ast.CallExpr, importPath string, names ...string) bool {
	return PkgRef(file, call.Fun, importPath, names...)
}

func PkgRef(file *ast.File, expr ast.Expr, importPath string, names ...string) bool {
	locals := ImportNames(file, importPath)
	// A parser-resolved identifier (Obj != nil) is a local declaration shadowing the import.
	switch typed := ast.Unparen(expr).(type) {
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		return ok && pkg.Obj == nil && slices.Contains(locals, pkg.Name) &&
			memberMatches(typed.Sel.Name, names)
	case *ast.Ident:
		return typed.Obj == nil && slices.Contains(locals, dotImport) &&
			len(names) > 0 && memberMatches(typed.Name, names)
	}
	return false
}

func memberMatches(member string, names []string) bool {
	return len(names) == 0 || slices.Contains(names, member)
}
