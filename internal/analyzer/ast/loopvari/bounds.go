package loopvari

import (
	"go/ast"
	"go/token"
)

func (s *scanner) isLenBound(expr ast.Expr) bool {
	call, ok := stripOffset(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	return ok && (fn.Name == "len" || fn.Name == "cap") && !s.shadowed[fn.Name]
}

func stripOffset(expr ast.Expr) ast.Expr {
	expr = unparen(expr)
	sum, ok := expr.(*ast.BinaryExpr)
	if !ok || (sum.Op != token.ADD && sum.Op != token.SUB) {
		return expr
	}
	switch {
	case isIntLiteral(sum.Y):
		return unparen(sum.X)
	case sum.Op == token.ADD && isIntLiteral(sum.X):
		return unparen(sum.Y)
	}
	return expr
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

func isIntLiteral(expr ast.Expr) bool {
	lit, ok := unparen(expr).(*ast.BasicLit)
	return ok && lit.Kind == token.INT
}
