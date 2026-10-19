package shadowing

import (
	"go/ast"
	"go/token"
)

type scopeWalker struct {
	scopes *scopeStack
	report reporter
}

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

func (walker scopeWalker) walkBlock(block *ast.BlockStmt) {
	if block == nil {
		return
	}
	walker.scopes.push()
	defer walker.scopes.pop()

	walker.walkStmts(block.List)
}

func (walker scopeWalker) walkStmts(stmts []ast.Stmt) {
	for _, stmt := range stmts {
		walker.walkStmt(stmt)
	}
}

func (walker scopeWalker) walkStmt(stmt ast.Stmt) {
	switch typed := stmt.(type) {
	case *ast.AssignStmt:
		walker.walkAssign(typed)
	case *ast.DeclStmt:
		walker.walkDecl(typed.Decl)
	case *ast.BlockStmt:
		walker.walkBlock(typed)
	case *ast.LabeledStmt:
		walker.walkStmt(typed.Stmt)
	case *ast.CaseClause:
		// Each clause is its own implicit block, so two clauses declaring the
		// same name are two declarations, not one plus a reassignment.
		walker.walkClause(nil, typed.Body)
	case *ast.CommClause:
		walker.walkClause(typed.Comm, typed.Body)
	default:
		walker.walkBranch(stmt)
	}
}

func (walker scopeWalker) walkBranch(stmt ast.Stmt) {
	switch typed := stmt.(type) {
	case *ast.IfStmt:
		walker.walkIf(typed)
	case *ast.ForStmt:
		walker.walkFor(typed)
	case *ast.RangeStmt:
		walker.walkRange(typed)
	case *ast.SwitchStmt:
		walker.walkSwitch(typed)
	case *ast.TypeSwitchStmt:
		walker.walkTypeSwitch(typed)
	case *ast.SelectStmt:
		walker.walkSelect(typed)
	}
}

func (walker scopeWalker) walkClause(
	comm ast.Stmt,
	body []ast.Stmt,
) {
	walker.scopes.push()
	defer walker.scopes.pop()

	if comm != nil {
		// `case v := <-ch:` binds v in the clause's own block.
		walker.walkStmt(comm)
	}
	walker.walkStmts(body)
}

func (walker scopeWalker) walkAssign(assign *ast.AssignStmt) {
	if assign.Tok != token.DEFINE {
		return
	}
	for _, lhs := range assign.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		walker.report.declaring(id, walker.scopes)
		walker.scopes.declare(id.Name)
	}
}

func (walker scopeWalker) walkDecl(decl ast.Decl) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		switch typed := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range typed.Names {
				walker.declareIdent(id)
			}
		case *ast.TypeSpec:
			walker.declareIdent(typed.Name)
		}
	}
}

func (walker scopeWalker) declareIfIdent(expr ast.Expr) {
	if id, ok := expr.(*ast.Ident); ok {
		walker.declareIdent(id)
	}
}

func (walker scopeWalker) declareIdent(id *ast.Ident) {
	if id == nil || id.Name == "_" {
		return
	}
	walker.report.declaring(id, walker.scopes)
	walker.scopes.declare(id.Name)
}
