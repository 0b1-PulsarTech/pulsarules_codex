package testfunc

import (
	"go/ast"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const KindExample = "Example"

func testPrefixes() []string {
	return []string{"Test", "Benchmark", "Fuzz", KindExample}
}

func Covers(fc pass.FileChange, file *ast.File, fn *ast.FuncDecl) bool {
	if !fc.IsTest {
		return false
	}
	if _, ok := Kind(file, fn); ok || isSuiteMethod(fn) {
		return true
	}
	return IsHelper(file, fn)
}

func Kind(file *ast.File, fn *ast.FuncDecl) (string, bool) {
	if fn.Recv != nil || fn.Name == nil {
		return "", false
	}
	alias := testingAlias(file)
	name := fn.Name.Name
	for _, prefix := range testPrefixes() {
		suffix, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		if suffix != "" && unicode.IsLower(rune(suffix[0])) {
			continue
		}
		if matchesSignature(name, prefix, alias, fn.Type) {
			return prefix, true
		}
	}
	return "", false
}

func isSuiteMethod(fn *ast.FuncDecl) bool {
	suffix, ok := strings.CutPrefix(fn.Name.Name, "Test")
	if fn.Recv == nil || !ok || (suffix != "" && unicode.IsLower(rune(suffix[0]))) {
		return false
	}
	return fn.Type.Results == nil && (fn.Type.Params == nil || len(fn.Type.Params.List) == 0)
}

func IsHelper(file *ast.File, fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}
	alias := testingAlias(file)
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if isTestingHandle(field.Type, alias) {
				return true
			}
		}
	}
	return callsHelper(fn.Body)
}

func isTestingReceiver(expr ast.Expr) bool {
	switch recv := expr.(type) {
	case *ast.SelectorExpr:
		return recv.Sel.Name == "t" || recv.Sel.Name == "T"
	case *ast.CallExpr:
		sel, ok := recv.Fun.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "T"
	}
	return false
}

func callsHelper(body *ast.BlockStmt) bool {
	return body != nil && goast.Any(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return false
		}
		sel, isSel := call.Fun.(*ast.SelectorExpr)
		return isSel && sel.Sel.Name == "Helper" && isTestingReceiver(sel.X)
	})
}
