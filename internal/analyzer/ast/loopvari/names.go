package loopvari

import (
	"go/ast"
	"go/token"
)

func isLoopIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == loopVarName
}

func mentionsLoopVar(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == loopVarName {
			found = true
		}
		return !found
	})
	return found
}

func isCounterValue(tok token.Token, rhs ast.Expr) bool {
	if tok != token.ASSIGN {
		return !mentionsLoopVar(rhs)
	}
	step, ok := rhs.(*ast.BinaryExpr)
	if !ok || (step.Op != token.ADD && step.Op != token.SUB) {
		return false
	}
	if isLoopIdent(step.X) {
		return !mentionsLoopVar(step.Y)
	}
	return step.Op == token.ADD && isLoopIdent(step.Y) && !mentionsLoopVar(step.X)
}

func definedAssign(stmt ast.Stmt) *ast.AssignStmt {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE || !hasLoopVar(assign.Lhs) {
		return nil
	}
	return assign
}

func loopVarDecl(stmt ast.Stmt) *ast.GenDecl {
	decl, ok := stmt.(*ast.DeclStmt)
	if !ok {
		return nil
	}
	gen, ok := decl.Decl.(*ast.GenDecl)
	if !ok {
		return nil
	}
	for _, spec := range gen.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok && namesLoopVar(valueSpec.Names) {
			return gen
		}
	}
	return nil
}

func namesLoopVar(idents []*ast.Ident) bool {
	for _, ident := range idents {
		if ident.Name == loopVarName {
			return true
		}
	}
	return false
}

func fieldsNameLoopVar(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		if namesLoopVar(field.Names) {
			return true
		}
	}
	return false
}

func exprNodes(exprs []ast.Expr) []ast.Node {
	nodes := make([]ast.Node, 0, len(exprs))
	for _, expr := range exprs {
		nodes = append(nodes, expr)
	}
	return nodes
}
