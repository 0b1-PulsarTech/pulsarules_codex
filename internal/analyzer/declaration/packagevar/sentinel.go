package packagevar

import (
	"go/ast"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func (scan fileScan) sentinelCall(call *ast.CallExpr, name string) bool {
	return (scan.isStdSentinel(call) || isErrName(name) && scan.isErrConstructor(call)) &&
		scan.simpleArgs(call.Args)
}

func (scan fileScan) nestedSentinel(call *ast.CallExpr) bool {
	return (scan.isStdSentinel(call) || scan.isErrConstructor(call)) && scan.simpleArgs(call.Args)
}

func (scan fileScan) isStdSentinel(call *ast.CallExpr) bool {
	return goast.PkgCall(scan.file, call, "errors", "New") ||
		goast.PkgCall(scan.file, call, "fmt", "Errorf")
}

func (scan fileScan) isErrConstructor(call *ast.CallExpr) bool {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		return isCtorName(fun.Name) && scan.errFuncs[fun.Name]
	case *ast.SelectorExpr:
		pkg, ok := fun.X.(*ast.Ident)
		return ok && scan.imports[pkg.Name] && isCtorName(fun.Sel.Name) &&
			strings.Contains(strings.ToLower(pkg.Name), "err")
	}
	return false
}

func isCtorName(callee string) bool {
	callee = strings.TrimPrefix(strings.TrimPrefix(callee, "New"), "new")
	rest, ok := strings.CutPrefix(callee, "Err")
	return ok && (rest == "" || isErrBoundary(rest[0]))
}

func (scan fileScan) simpleArgs(args []ast.Expr) bool {
	for _, arg := range args {
		if !scan.isSimpleArg(arg) {
			return false
		}
	}
	return true
}

func (scan fileScan) isSimpleArg(expr ast.Expr) bool {
	switch arg := ast.Unparen(expr).(type) {
	case *ast.BasicLit, *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return scan.isSimpleArg(arg.X)
	case *ast.BinaryExpr:
		return scan.isSimpleArg(arg.X) && scan.isSimpleArg(arg.Y)
	case *ast.CallExpr:
		return scan.nestedSentinel(arg)
	}
	return false
}

func isErrReturning(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Type.Results == nil || fn.Type.Results.NumFields() != 1 {
		return false
	}
	typ := fn.Type.Results.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	ident, ok := typ.(*ast.Ident)
	return ok && (ident.Name == "error" || hasErrSuffix(ident.Name))
}
