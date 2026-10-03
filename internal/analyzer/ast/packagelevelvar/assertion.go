package packagelevelvar

import (
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"strconv"
)

func (scan fileScan) isAssertion(vs *ast.ValueSpec) bool {
	if len(vs.Values) == 0 {
		return vs.Type != nil
	}
	return scan.allAssertive(vs)
}

func (scan fileScan) allAssertive(vs *ast.ValueSpec) bool {
	for _, value := range vs.Values {
		if !scan.isConversionOrComposite(value) {
			return false
		}
	}
	return true
}

func (scan fileScan) isConversionOrComposite(expr ast.Expr) bool {
	switch value := unparen(expr).(type) {
	case *ast.CompositeLit:
		return scan.callFree(value)
	case *ast.Ident:
		return value.Name == "nil"
	case *ast.UnaryExpr:
		return value.Op == token.AND && scan.isConversionOrComposite(value.X)
	case *ast.CallExpr:
		return len(value.Args) == 1 && scan.isAssertCall(value)
	}
	return false
}

func (scan fileScan) isAssertCall(call *ast.CallExpr) bool {
	arg := unparen(call.Args[0])
	switch fun := unparen(call.Fun).(type) {
	case *ast.Ident:
		if fun.Name == "new" {
			return scan.isTypeExpr(arg)
		}
	case *ast.SelectorExpr:
		return scan.isImportedSel(fun) && isNil(arg)
	}
	return scan.isConversionType(call.Fun) &&
		(isLiteralArg(arg) || scan.isConversionOrComposite(arg))
}

func (scan fileScan) callFree(lit *ast.CompositeLit) bool {
	free := true
	ast.Inspect(lit, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			isConv := len(typed.Args) == 1 && scan.isConversionType(typed.Fun)
			free = free && isConv
		}
		return free
	})
	return free
}

func (scan fileScan) isConversionType(fun ast.Expr) bool {
	switch typ := fun.(type) {
	case *ast.Ident:
		return scan.isTypeName(typ.Name)
	case *ast.ParenExpr:
		inner := unparen(typ.X)
		if star, isStar := inner.(*ast.StarExpr); isStar {
			return scan.isTypeExpr(star.X)
		}
		return isTypeLiteral(inner)
	case *ast.IndexExpr, *ast.IndexListExpr:
		return scan.isTypeExpr(typ)
	}
	return isTypeLiteral(fun)
}

func (scan fileScan) isTypeExpr(expr ast.Expr) bool {
	inner := unparen(expr)
	switch typ := inner.(type) {
	case *ast.Ident:
		return scan.isTypeName(typ.Name)
	case *ast.SelectorExpr:
		return scan.isImportedSel(typ)
	case *ast.StarExpr:
		return scan.isTypeExpr(typ.X)
	case *ast.IndexExpr:
		return scan.isTypeExpr(typ.X)
	case *ast.IndexListExpr:
		return scan.isTypeExpr(typ.X)
	}
	return isTypeLiteral(inner)
}

func isLiteralArg(expr ast.Expr) bool {
	if sign, isSigned := expr.(*ast.UnaryExpr); isSigned && isSignOp(sign.Op) {
		expr = unparen(sign.X)
	}
	_, isLit := expr.(*ast.BasicLit)
	return isLit
}

func isSignOp(op token.Token) bool {
	return op == token.SUB || op == token.ADD
}

func isNil(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func isTypeLiteral(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.FuncType,
		*ast.InterfaceType, *ast.StructType:
		return true
	}
	return false
}

func (scan fileScan) isTypeName(name string) bool {
	if scan.types[name] {
		return true
	}
	_, isType := types.Universe.Lookup(name).(*types.TypeName)
	return isType
}

func importNames(file *ast.File) map[string]bool {
	names := make(map[string]bool, len(file.Imports))
	for _, spec := range file.Imports {
		got, err := strconv.Unquote(spec.Path.Value)
		switch {
		case err != nil:
		case spec.Name == nil:
			names[path.Base(got)] = true
		case spec.Name.Name != "_" && spec.Name.Name != ".":
			names[spec.Name.Name] = true
		}
	}
	return names
}

func (scan fileScan) isImportedSel(sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	return ok && scan.imports[ident.Name]
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = paren.X
	}
}
