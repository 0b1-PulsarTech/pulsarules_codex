package variadicindex

import (
	"go/ast"
	"go/token"
)

func (walker scope) isTarget(expr ast.Expr) bool {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == walker.name
}

func (walker scope) hasField(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		if walker.hasName(field.Names) {
			return true
		}
	}
	return false
}

func (walker scope) hasName(idents []*ast.Ident) bool {
	for _, ident := range idents {
		if ident.Name == walker.name {
			return true
		}
	}
	return false
}

func (walker scope) hasExpr(exprs []ast.Expr) bool {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == walker.name {
			return true
		}
	}
	return false
}

func (walker scope) initShadows(init ast.Stmt) bool {
	assign, ok := init.(*ast.AssignStmt)
	if !ok || !walker.defines(assign) {
		walker.node(init)
		return false
	}
	walker.exprs(assign.Rhs)
	return true
}

func (walker scope) defines(assign *ast.AssignStmt) bool {
	return assign.Tok == token.DEFINE && walker.hasExpr(assign.Lhs)
}

func (walker scope) exprs(list []ast.Expr) {
	for _, expr := range list {
		walker.node(expr)
	}
}
