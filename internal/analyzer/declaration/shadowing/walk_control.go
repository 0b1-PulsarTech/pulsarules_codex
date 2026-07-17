package shadowing

import (
	"go/ast"
	"go/token"
)

func (a *Analyzer) walkIf(
	s *ast.IfStmt,
	scopes *scopeStack,
	report reporter,
) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, report)
	}
	a.walkBlock(s.Body, scopes, report)
	if s.Else != nil {
		a.walkStmt(s.Else, scopes, report)
	}
	scopes.pop()
}

func (a *Analyzer) walkFor(
	s *ast.ForStmt,
	scopes *scopeStack,
	report reporter,
) {
	// The braces open a further nested block, so this scope and the body's are both pushed.
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, report)
	}
	a.walkBlock(s.Body, scopes, report)
	scopes.pop()
}

func (a *Analyzer) walkRange(
	s *ast.RangeStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if s.Tok == token.DEFINE {
		declareIfIdent(s.Key, scopes, report)
		declareIfIdent(s.Value, scopes, report)
	}
	a.walkBlock(s.Body, scopes, report)
	scopes.pop()
}

func (a *Analyzer) walkSwitch(
	s *ast.SwitchStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, report)
	}
	a.walkClauses(s.Body, scopes, report)
	scopes.pop()
}

func (a *Analyzer) walkTypeSwitch(
	s *ast.TypeSwitchStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, report)
	}
	if as, ok := s.Assign.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		if id, ok := as.Lhs[0].(*ast.Ident); ok {
			// The guard binds a name like any other declaration, so it goes
			// through declareIdent and gets the same shadow check.
			declareIdent(id, scopes, report)
		}
	}
	a.walkClauses(s.Body, scopes, report)
	scopes.pop()
}

func (a *Analyzer) walkSelect(
	s *ast.SelectStmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	a.walkClauses(s.Body, scopes, report)
	scopes.pop()
}

func (a *Analyzer) walkClauses(
	body *ast.BlockStmt,
	scopes *scopeStack,
	report reporter,
) {
	if body == nil {
		return
	}
	// The braces around clauses are not a block: statements run in this scope
	// and each clause pushes its own.
	a.walkStmts(body.List, scopes, report)
}
