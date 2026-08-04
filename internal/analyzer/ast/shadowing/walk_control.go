package shadowing

import (
	"go/ast"
	"go/token"
)

// Each if/for/switch/select is its own implicit block. For if and for, the
// braces open a further nested block, so both get pushed and a `:=` in the
// body genuinely shadows the init statement's binding. A switch or select has
// no such intermediate block: its braces hold clauses, and it is each CLAUSE
// that is the implicit block (see walkClause).

func (a *Analyzer) walkIf(
	s *ast.IfStmt,
	scopes *scopeStack,
	report reporter,
) {
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

// walkClauses walks a switch or select body. The braces around the clauses are
// not a block of their own, so the statements run in the current scope and
// each clause pushes its own.
func (a *Analyzer) walkClauses(
	body *ast.BlockStmt,
	scopes *scopeStack,
	report reporter,
) {
	if body == nil {
		return
	}
	a.walkStmts(body.List, scopes, report)
}
