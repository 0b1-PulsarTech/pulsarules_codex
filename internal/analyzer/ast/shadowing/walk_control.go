package shadowing

import (
	"go/ast"
	"go/token"
)

func (analyzer *Analyzer) walkIf(
	ifStmt *ast.IfStmt,
	scopes *scopeStack,
	report reporter,
) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	scopes.push()
	if ifStmt.Init != nil {
		analyzer.walkStmt(ifStmt.Init, scopes, report)
	}
	analyzer.walkBlock(ifStmt.Body, scopes, report)
	if ifStmt.Else != nil {
		analyzer.walkStmt(ifStmt.Else, scopes, report)
	}
	scopes.pop()
}

func (analyzer *Analyzer) walkFor(
	forStmt *ast.ForStmt,
	scopes *scopeStack,
	report reporter,
) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	scopes.push()
	if forStmt.Init != nil {
		analyzer.walkStmt(forStmt.Init, scopes, report)
	}
	analyzer.walkBlock(forStmt.Body, scopes, report)
	scopes.pop()
}

func (analyzer *Analyzer) walkRange(
	rangeStmt *ast.RangeStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if rangeStmt.Tok == token.DEFINE {
		declareIfIdent(rangeStmt.Key, scopes, report)
		declareIfIdent(rangeStmt.Value, scopes, report)
	}
	analyzer.walkBlock(rangeStmt.Body, scopes, report)
	scopes.pop()
}

func (analyzer *Analyzer) walkSwitch(
	switchStmt *ast.SwitchStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if switchStmt.Init != nil {
		analyzer.walkStmt(switchStmt.Init, scopes, report)
	}
	analyzer.walkClauses(switchStmt.Body, scopes, report)
	scopes.pop()
}

func (analyzer *Analyzer) walkTypeSwitch(
	typeSwitch *ast.TypeSwitchStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if typeSwitch.Init != nil {
		analyzer.walkStmt(typeSwitch.Init, scopes, report)
	}
	if as, ok := typeSwitch.Assign.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			// The guard binds a name like any other declaration, so it goes
			// through declareIdent and gets the same shadow check.
			declareIdent(id, scopes, report)
		}
	}
	analyzer.walkClauses(typeSwitch.Body, scopes, report)
	scopes.pop()
}

func (analyzer *Analyzer) walkSelect(
	selectStmt *ast.SelectStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	analyzer.walkClauses(selectStmt.Body, scopes, report)
	scopes.pop()
}

func (analyzer *Analyzer) walkClauses(
	body *ast.BlockStmt,
	scopes *scopeStack,
	report reporter,
) {
	if body == nil {
		return
	}
	// The braces around clauses are not a block: statements run in this scope
	// and each clause pushes its own.
	analyzer.walkStmts(body.List, scopes, report)
}
