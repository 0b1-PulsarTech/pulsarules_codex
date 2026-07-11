package shadowing

import (
	"go/ast"
	"go/token"
)

func (a *Analyzer) walkIf(
	s *ast.IfStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, emit)
	}
	a.walkBlock(s.Body, scopes, emit)
	if s.Else != nil {
		a.walkStmt(s.Else, scopes, emit)
	}
	scopes.pop()
}

func (a *Analyzer) walkFor(
	s *ast.ForStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, emit)
	}
	a.walkBlock(s.Body, scopes, emit)
	scopes.pop()
}

func (a *Analyzer) walkRange(
	s *ast.RangeStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	if s.Tok == token.DEFINE {
		a.declareIfIdent(s.Key, scopes, emit)
		a.declareIfIdent(s.Value, scopes, emit)
	}
	a.walkBlock(s.Body, scopes, emit)
	scopes.pop()
}

func (a *Analyzer) walkSwitch(
	s *ast.SwitchStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, emit)
	}
	a.walkBlock(s.Body, scopes, emit)
	scopes.pop()
}

func (a *Analyzer) walkTypeSwitch(
	s *ast.TypeSwitchStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	if s.Init != nil {
		a.walkStmt(s.Init, scopes, emit)
	}
	if as, ok := s.Assign.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		if id, ok := as.Lhs[0].(*ast.Ident); ok && id.Name != "_" {
			scopes.declare(id.Name, id.NamePos)
		}
	}
	a.walkBlock(s.Body, scopes, emit)
	scopes.pop()
}

func (a *Analyzer) walkSelect(
	s *ast.SelectStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	scopes.push()
	a.walkBlock(s.Body, scopes, emit)
	scopes.pop()
}
