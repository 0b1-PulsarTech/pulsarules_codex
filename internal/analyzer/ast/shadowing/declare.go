package shadowing

import (
	"go/ast"
	"go/token"
)

func (a *Analyzer) declareIfIdent(
	expr ast.Expr,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	id, ok := expr.(*ast.Ident)
	if !ok || id.Name == "_" {
		return
	}
	if hidden, ok := scopes.lookup(id.Name); ok {
		emit(id.NamePos, id.Name, hidden)
	}
	scopes.declare(id.Name, id.NamePos)
}
