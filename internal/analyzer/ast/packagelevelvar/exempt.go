package packagelevelvar

import (
	"go/ast"
	"go/token"
	"strings"
)

func (scan fileScan) isSpecExempt(vs *ast.ValueSpec, idx int) bool {
	if len(vs.Values) == 0 {
		return false
	}
	if len(vs.Names) != len(vs.Values) {
		return scan.allValuesExempt(vs)
	}
	return scan.isValueExempt(vs.Type, vs.Names[idx].Name, vs.Values[idx])
}

func (scan fileScan) allValuesExempt(vs *ast.ValueSpec) bool {
	for _, value := range vs.Values {
		if !scan.isValueExempt(vs.Type, "", value) {
			return false
		}
	}
	return true
}

func (scan fileScan) isValueExempt(typ ast.Expr, name string, value ast.Expr) bool {
	if isErrorType(typ) {
		return scan.isShapedError(name, value)
	}
	return scan.isSentinelValue(name, value)
}

func (scan fileScan) isShapedError(name string, expr ast.Expr) bool {
	switch value := unparen(expr).(type) {
	case *ast.Ident:
		return name == "_" && value.Name == "nil"
	case *ast.CompositeLit:
		return isErrName(name) && scan.callFree(value)
	case *ast.UnaryExpr:
		return isErrName(name) && scan.isAddrOfCallFreeLit(value)
	case *ast.SelectorExpr:
		return scan.isImportedSel(value) && isErrName(name)
	}
	return scan.isSentinelValue(name, expr)
}

func (scan fileScan) isAddrOfCallFreeLit(value *ast.UnaryExpr) bool {
	lit, isLit := unparen(value.X).(*ast.CompositeLit)
	return value.Op == token.AND && isLit && scan.callFree(lit)
}

func isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "error"
}

func (scan fileScan) isImportedSel(sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	return ok && scan.imports[ident.Name]
}

func (scan fileScan) isSentinelValue(name string, expr ast.Expr) bool {
	switch value := unparen(expr).(type) {
	case *ast.CallExpr:
		return scan.sentinelCall(value, name)
	case *ast.SelectorExpr:
		return scan.isImportedSel(value) && isErrSelector(value.Sel.Name) && isErrName(name)
	case *ast.CompositeLit:
		return scan.isErrComposite(name, value)
	case *ast.UnaryExpr:
		lit, isLit := unparen(value.X).(*ast.CompositeLit)
		return value.Op == token.AND && isLit && scan.isErrComposite(name, lit)
	}
	return false
}

func (scan fileScan) isErrComposite(name string, lit *ast.CompositeLit) bool {
	return isErrName(name) && isErrTypeName(lit.Type) && scan.callFree(lit)
}

func isErrTypeName(expr ast.Expr) bool {
	var typeName string
	switch typ := expr.(type) {
	case *ast.Ident:
		typeName = typ.Name
	case *ast.SelectorExpr:
		typeName = typ.Sel.Name
	}
	return hasErrSuffix(typeName)
}

func hasErrSuffix(name string) bool {
	return name == "err" || strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "Error")
}

func errTail(name string) (string, bool) {
	if rest, ok := strings.CutPrefix(name, "Err"); ok {
		return rest, true
	}
	return strings.CutPrefix(name, "err")
}

func isErrSelector(name string) bool {
	rest, ok := errTail(name)
	return ok && (rest == "" || isErrBoundary(rest[0]))
}

func isErrName(name string) bool {
	rest, ok := errTail(name)
	return ok && rest != "" && isErrBoundary(rest[0])
}

func isErrBoundary(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_'
}
