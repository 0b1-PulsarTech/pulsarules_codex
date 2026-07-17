package shadowing

import (
	"go/ast"
	"go/token"
)

type reporter struct {
	shadow func(pos token.Pos, name, kind string)
	reuse  func(pos token.Pos, name string)
}

func (report reporter) declaring(id *ast.Ident, scopes *scopeStack) {
	if origin, ok := scopes.declaredHere(id.Name); ok {
		// Legal and silent unless the name came from the signature; an
		// ordinary `a, err := f()` then `b, err := g()` must stay quiet.
		if origin == originSignature {
			report.reuse(id.NamePos, id.Name)
		}
		return
	}
	if hidden, ok := scopes.lookup(id.Name); ok {
		report.shadow(id.NamePos, id.Name, shadowKind(hidden))
	}
}

func shadowKind(origin string) string {
	// The signature marker is internal: to a reader it is just a variable.
	if origin == originSignature {
		return originVariable
	}
	return origin
}

func (a *Analyzer) walkBlock(
	block *ast.BlockStmt,
	scopes *scopeStack,
	report reporter,
) {
	if block == nil {
		return
	}
	scopes.push()
	defer scopes.pop()

	a.walkStmts(block.List, scopes, report)
}

func (a *Analyzer) walkStmts(
	stmts []ast.Stmt,
	scopes *scopeStack,
	report reporter,
) {
	for _, stmt := range stmts {
		a.walkStmt(stmt, scopes, report)
	}
}

func (a *Analyzer) walkStmt(
	stmt ast.Stmt,
	scopes *scopeStack,
	report reporter,
) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		a.walkAssign(s, scopes, report)
	case *ast.DeclStmt:
		a.walkDecl(s.Decl, scopes, report)
	case *ast.IfStmt:
		a.walkIf(s, scopes, report)
	case *ast.ForStmt:
		a.walkFor(s, scopes, report)
	case *ast.RangeStmt:
		a.walkRange(s, scopes, report)
	case *ast.SwitchStmt:
		a.walkSwitch(s, scopes, report)
	case *ast.TypeSwitchStmt:
		a.walkTypeSwitch(s, scopes, report)
	case *ast.SelectStmt:
		a.walkSelect(s, scopes, report)
	case *ast.BlockStmt:
		a.walkBlock(s, scopes, report)
	case *ast.LabeledStmt:
		a.walkStmt(s.Stmt, scopes, report)
	case *ast.CaseClause:
		// Each clause is its own implicit block, so two clauses declaring the
		// same name are two declarations, not one plus a reassignment.
		a.walkClause(nil, s.Body, scopes, report)
	case *ast.CommClause:
		a.walkClause(s.Comm, s.Body, scopes, report)
	}
}

func (a *Analyzer) walkClause(
	comm ast.Stmt,
	body []ast.Stmt,
	scopes *scopeStack,
	report reporter,
) {
	scopes.push()
	defer scopes.pop()

	if comm != nil {
		// `case v := <-ch:` binds v in the clause's own block.
		a.walkStmt(comm, scopes, report)
	}
	a.walkStmts(body, scopes, report)
}

func (a *Analyzer) walkAssign(
	s *ast.AssignStmt,
	scopes *scopeStack,
	report reporter,
) {
	if s.Tok != token.DEFINE {
		return
	}
	for _, lhs := range s.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		report.declaring(id, scopes)
		scopes.declare(id.Name)
	}
}

func (a *Analyzer) walkDecl(
	decl ast.Decl,
	scopes *scopeStack,
	report reporter,
) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range s.Names {
				declareIdent(id, scopes, report)
			}
		case *ast.TypeSpec:
			declareIdent(s.Name, scopes, report)
		}
	}
}

func declareIfIdent(expr ast.Expr, scopes *scopeStack, report reporter) {
	if id, ok := expr.(*ast.Ident); ok {
		declareIdent(id, scopes, report)
	}
}

func declareIdent(id *ast.Ident, scopes *scopeStack, report reporter) {
	if id == nil || id.Name == "_" {
		return
	}
	report.declaring(id, scopes)
	scopes.declare(id.Name)
}
