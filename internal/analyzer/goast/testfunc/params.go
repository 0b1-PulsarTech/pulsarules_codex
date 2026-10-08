package testfunc

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func testingAlias(file *ast.File) string {
	const testingImportPath = "testing"
	if file == nil {
		return testingImportPath
	}
	if names := goast.ImportNames(file, testingImportPath); len(names) > 0 {
		return names[0]
	}
	return testingImportPath
}

func qualified(expr ast.Expr, alias string) (pkg, name string, isPointer bool) {
	const dotImport = "."
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
		isPointer = true
	}
	if ident, ok := expr.(*ast.Ident); ok && alias == dotImport {
		return dotImport, ident.Name, isPointer
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	return ident.Name, sel.Sel.Name, isPointer
}

func isTestingHandle(expr ast.Expr, alias string) bool {
	pkg, name, isPointer := qualified(expr, alias)
	switch {
	case name == "TestingT":
		return !isPointer
	case pkg != alias:
		return false
	case name == "TB":
		return !isPointer
	default:
		return isPointer && (name == "T" || name == "B" || name == "F")
	}
}

func wantParam(name, prefix string) string {
	if name == "TestMain" {
		return "M"
	}
	switch prefix {
	case "Test":
		return "T"
	case "Benchmark":
		return "B"
	case "Fuzz":
		return "F"
	default:
		return ""
	}
}

func matchesSignature(name, prefix, alias string, ft *ast.FuncType) bool {
	want := wantParam(name, prefix)
	if want == "" {
		return ft.Params == nil || len(ft.Params.List) == 0
	}
	if ft.Params == nil || len(ft.Params.List) != 1 {
		return false
	}
	pkg, got, isPointer := qualified(ft.Params.List[0].Type, alias)
	return isPointer && pkg == alias && got == want
}
