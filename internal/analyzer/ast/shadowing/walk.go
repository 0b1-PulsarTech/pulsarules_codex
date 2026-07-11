package shadowing

import (
	"go/ast"
	"go/token"
)

func (a *Analyzer) walkBlock(
	block *ast.BlockStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	if block == nil {
		return
	}
	scopes.push()
	defer scopes.pop()

	for _, stmt := range block.List {
		a.walkStmt(stmt, scopes, emit)
	}
}

func (a *Analyzer) walkStmt(
	stmt ast.Stmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		a.walkAssign(s, scopes, emit)
	case *ast.DeclStmt:
		a.walkDecl(s.Decl, scopes, emit)
	case *ast.IfStmt:
		a.walkIf(s, scopes, emit)
	case *ast.ForStmt:
		a.walkFor(s, scopes, emit)
	case *ast.RangeStmt:
		a.walkRange(s, scopes, emit)
	case *ast.SwitchStmt:
		a.walkSwitch(s, scopes, emit)
	case *ast.TypeSwitchStmt:
		a.walkTypeSwitch(s, scopes, emit)
	case *ast.SelectStmt:
		a.walkSelect(s, scopes, emit)
	case *ast.BlockStmt:
		a.walkBlock(s, scopes, emit)
	case *ast.LabeledStmt:
		a.walkStmt(s.Stmt, scopes, emit)
	case *ast.CaseClause:
		for _, b := range s.Body {
			a.walkStmt(b, scopes, emit)
		}
	case *ast.CommClause:
		for _, b := range s.Body {
			a.walkStmt(b, scopes, emit)
		}
	}
}

func (a *Analyzer) walkAssign(
	s *ast.AssignStmt,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	if s.Tok != token.DEFINE {
		return
	}
	for _, lhs := range s.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		if hidden, ok := scopes.lookup(id.Name); ok {
			emit(id.NamePos, id.Name, hidden)
		}
		scopes.declare(id.Name, id.NamePos)
	}
}

func (a *Analyzer) walkDecl(
	decl ast.Decl,
	scopes *scopeStack,
	emit func(token.Pos, string, string),
) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range s.Names {
				if id.Name == "_" {
					continue
				}
				if hidden, ok := scopes.lookup(id.Name); ok {
					emit(id.NamePos, id.Name, hidden)
				}
				scopes.declare(id.Name, id.NamePos)
			}
		case *ast.TypeSpec:
			if s.Name == nil || s.Name.Name == "_" {
				continue
			}
			if hidden, ok := scopes.lookup(s.Name.Name); ok {
				emit(s.Name.NamePos, s.Name.Name, hidden)
			}
			scopes.declare(s.Name.Name, s.Name.NamePos)
		}
	}
}
