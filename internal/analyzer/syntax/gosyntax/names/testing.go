package names

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func testingNames(file *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, name := range goast.ImportNames(file, "testing") {
		aliases[name] = true
	}
	return aliases
}

func testingTypeName(expr ast.Expr, aliases map[string]bool) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		if aliases["."] {
			return typed.Name
		}
	case *ast.SelectorExpr:
		pkg, ok := typed.X.(*ast.Ident)
		if ok && aliases[pkg.Name] {
			return typed.Sel.Name
		}
	}
	return ""
}

func isTestingType(expr ast.Expr, aliases map[string]bool) bool {
	expr = ast.Unparen(expr)
	star, isPointer := expr.(*ast.StarExpr)
	if isPointer {
		expr = ast.Unparen(star.X)
	}
	switch testingTypeName(expr, aliases) {
	case "T", "B", "F", "M":
		return isPointer
	case "TB":
		return !isPointer
	}
	return false
}
