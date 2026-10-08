package names

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func testingNames(file *ast.File) map[string]bool {
	names := map[string]bool{}
	for _, name := range goast.ImportNames(file, "testing") {
		names[name] = true
	}
	return names
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
	expr = ast.Unparen(expr)
	star, isPointer := expr.(*ast.StarExpr)
	if isPointer {
		expr = ast.Unparen(star.X)
	}
	switch testingTypeName(expr, names) {
	case "T", "B", "F", "M":
		return isPointer
	case "TB":
		return !isPointer
	}
	return false
}
