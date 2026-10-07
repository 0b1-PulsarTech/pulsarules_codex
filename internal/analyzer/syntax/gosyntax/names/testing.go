package names

import (
	"go/ast"
	"strconv"
)

func testingNames(file *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, spec := range file.Imports {
		if path, _ := strconv.Unquote(spec.Path.Value); path != "testing" {
			continue
		}
		if spec.Name == nil {
			names["testing"] = true
			continue
		}
		names[spec.Name.Name] = true
	}
	return names
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, isParen := expr.(*ast.ParenExpr)
		if !isParen {
			return expr
		}
		expr = paren.X
	}
}

func testingTypeName(expr ast.Expr, names map[string]bool) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		if names["."] {
			return typed.Name
		}
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		if ok && names[pkg.Name] {
			return typed.Sel.Name
		}
	}
	return ""
}

func isTestingType(expr ast.Expr, names map[string]bool) bool {
	expr = unparen(expr)
	star, isPointer := expr.(*ast.StarExpr)
	if isPointer {
		expr = unparen(star.X)
	}
	switch testingTypeName(expr, names) {
	case "T", "B", "F", "M":
		return isPointer
	case "TB":
		return !isPointer
	}
	return false
}
