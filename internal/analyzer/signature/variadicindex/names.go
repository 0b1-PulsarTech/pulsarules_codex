package variadicindex

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func (walker scope) isTarget(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && ident.Name == walker.name
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
	return assign.Tok == token.DEFINE && goast.ExprsMention(assign.Lhs, walker.name)
}

func (walker scope) exprs(list []ast.Expr) {
	for _, expr := range list {
		walker.node(expr)
	}
}
