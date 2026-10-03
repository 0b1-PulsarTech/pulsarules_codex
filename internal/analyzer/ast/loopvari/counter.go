package loopvari

import (
	"go/ast"
	"go/token"
	"slices"
)

func countsInner(loop *ast.ForStmt, bound ast.Expr) bool {
	ident, ok := stripOffset(bound).(*ast.Ident)
	if !ok || ident.Name == loopVarName {
		return false
	}
	return movesName(loop, ident.Name)
}

func movesName(root ast.Node, name string) bool {
	var moves bool
	ast.Inspect(root, func(n ast.Node) bool {
		if moves || (n != root && rebinds(n, name)) {
			return false
		}
		switch node := n.(type) {
		case *ast.IncDecStmt:
			moves = isNamed(node.X, name)
		case *ast.AssignStmt:
			moves = node.Tok != token.DEFINE && slices.ContainsFunc(
				node.Lhs, func(lhs ast.Expr) bool { return isNamed(lhs, name) },
			)
		case *ast.BlockStmt:
			moves = movesInList(node.List, name)
			return false
		case *ast.CaseClause:
			moves = exprsMove(node.List, name) || movesInList(node.Body, name)
			return false
		case *ast.CommClause:
			moves = commMoves(node, name)
			return false
		}
		return true
	})
	return moves
}

func movesInList(stmts []ast.Stmt, name string) bool {
	for _, stmt := range stmts {
		if values, defines := definition(stmt, name); defines {
			return exprsMove(values, name)
		}
		if !rebinds(stmt, name) && movesName(stmt, name) {
			return true
		}
	}
	return false
}

func commMoves(clause *ast.CommClause, name string) bool {
	if values, defines := definition(clause.Comm, name); defines {
		return exprsMove(values, name)
	}
	return (clause.Comm != nil && movesName(clause.Comm, name)) || movesInList(clause.Body, name)
}

func exprsMove(exprs []ast.Expr, name string) bool {
	return slices.ContainsFunc(exprs, func(expr ast.Expr) bool {
		return !rebinds(expr, name) && movesName(expr, name)
	})
}

func rebinds(n ast.Node, name string) bool {
	switch node := n.(type) {
	case *ast.ForStmt:
		return redeclares(node.Init, name)
	case *ast.IfStmt:
		return redeclares(node.Init, name)
	case *ast.SwitchStmt:
		return redeclares(node.Init, name)
	case *ast.TypeSwitchStmt:
		return redeclares(node.Init, name) || redeclares(node.Assign, name)
	case *ast.RangeStmt:
		return node.Tok == token.DEFINE && (isNamed(node.Key, name) || isNamed(node.Value, name))
	case *ast.FuncLit:
		return hasName(funcFields(node.Type, nil), name)
	}
	return false
}

func definition(stmt ast.Stmt, name string) (values []ast.Expr, defines bool) {
	if assign, ok := stmt.(*ast.AssignStmt); ok && redeclares(assign, name) {
		return assign.Rhs, true
	}
	decl, ok := stmt.(*ast.DeclStmt)
	if !ok {
		return nil, false
	}
	gen, ok := decl.Decl.(*ast.GenDecl)
	if !ok {
		return nil, false
	}
	for _, spec := range gen.Specs {
		value, isValue := spec.(*ast.ValueSpec)
		if !isValue {
			continue
		}
		values = append(values, value.Values...)
		if hasName(value.Names, name) {
			return values, true
		}
	}
	return nil, false
}

func redeclares(stmt ast.Stmt, name string) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	return ok && assign.Tok == token.DEFINE &&
		slices.ContainsFunc(assign.Lhs, func(lhs ast.Expr) bool { return isNamed(lhs, name) })
}

func hasName(idents []*ast.Ident, name string) bool {
	return slices.ContainsFunc(idents, func(id *ast.Ident) bool { return id.Name == name })
}

func isNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}
