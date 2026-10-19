package shadowing

import (
	"go/ast"
	"go/token"
)

func (walker scopeWalker) walkIf(ifStmt *ast.IfStmt) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	walker.scopes.push()
	if ifStmt.Init != nil {
		walker.walkStmt(ifStmt.Init)
	}
	walker.walkBlock(ifStmt.Body)
	if ifStmt.Else != nil {
		walker.walkStmt(ifStmt.Else)
	}
	walker.scopes.pop()
}

func (walker scopeWalker) walkFor(forStmt *ast.ForStmt) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	walker.scopes.push()
	if forStmt.Init != nil {
		walker.walkStmt(forStmt.Init)
	}
	walker.walkBlock(forStmt.Body)
	walker.scopes.pop()
}

func (walker scopeWalker) walkRange(rangeStmt *ast.RangeStmt) {
	walker.scopes.push()
	if rangeStmt.Tok == token.DEFINE {
		walker.declareIfIdent(rangeStmt.Key)
		walker.declareIfIdent(rangeStmt.Value)
	}
	walker.walkBlock(rangeStmt.Body)
	walker.scopes.pop()
}

func (walker scopeWalker) walkSwitch(switchStmt *ast.SwitchStmt) {
	walker.scopes.push()
	if switchStmt.Init != nil {
		walker.walkStmt(switchStmt.Init)
	}
	walker.walkClauses(switchStmt.Body)
	walker.scopes.pop()
}

func (walker scopeWalker) walkTypeSwitch(typeSwitch *ast.TypeSwitchStmt) {
	walker.scopes.push()
	if typeSwitch.Init != nil {
		walker.walkStmt(typeSwitch.Init)
	}
	if as, ok := typeSwitch.Assign.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			// The guard binds a name like any other declaration, so it goes
			// through declareIdent and gets the same shadow check.
			walker.declareIdent(id)
		}
	}
	walker.walkClauses(typeSwitch.Body)
	walker.scopes.pop()
}

func (walker scopeWalker) walkSelect(selectStmt *ast.SelectStmt) {
	walker.scopes.push()
	walker.walkClauses(selectStmt.Body)
	walker.scopes.pop()
}

func (walker scopeWalker) walkClauses(body *ast.BlockStmt) {
	if body == nil {
		return
	}
	// The braces around clauses are not a block: statements run in this scope
	// and each clause pushes its own.
	walker.walkStmts(body.List)
}
