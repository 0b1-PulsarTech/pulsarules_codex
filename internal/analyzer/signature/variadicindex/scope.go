package variadicindex

import (
	"go/ast"
	"go/token"
)

type scope struct {
	name   string
	report func(*ast.IndexExpr, string)
}

func (walker scope) topStmts(list []ast.Stmt) {
	for _, stmt := range list {
		walker.stmt(stmt)
	}
}

func (walker scope) stmts(list []ast.Stmt) {
	for _, stmt := range list {
		if walker.shadowsRest(stmt) {
			return
		}
		walker.stmt(stmt)
	}
}

func (walker scope) shadowsRest(stmt ast.Stmt) bool {
	switch node := stmt.(type) {
	case *ast.AssignStmt:
		if !walker.defines(node) {
			return false
		}
		walker.exprs(node.Rhs)
		return true
	case *ast.DeclStmt:
		return walker.declares(node)
	}
	return false
}

func (walker scope) declares(decl *ast.DeclStmt) bool {
	gen, ok := decl.Decl.(*ast.GenDecl)
	if !ok {
		return false
	}
	for _, spec := range gen.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		walker.exprs(valueSpec.Values)
		if walker.hasName(valueSpec.Names) {
			return true
		}
	}
	return false
}

func (walker scope) stmt(stmt ast.Stmt) {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		walker.stmts(node.List)
	case *ast.IfStmt:
		walker.ifStmt(node)
	case *ast.ForStmt:
		walker.forStmt(node)
	case *ast.RangeStmt:
		walker.rangeStmt(node)
	case *ast.SwitchStmt:
		walker.switchStmt(node)
	case *ast.TypeSwitchStmt:
		walker.typeSwitch(node)
	case *ast.SelectStmt:
		walker.clauses(node.Body)
	case *ast.LabeledStmt:
		walker.stmt(node.Stmt)
	default:
		walker.node(stmt)
	}
}

func (walker scope) ifStmt(node *ast.IfStmt) {
	if walker.initShadows(node.Init) {
		return
	}
	walker.node(node.Cond)
	walker.stmts(node.Body.List)
	if node.Else != nil {
		walker.stmt(node.Else)
	}
}

func (walker scope) forStmt(node *ast.ForStmt) {
	if walker.initShadows(node.Init) {
		return
	}
	walker.node(node.Cond)
	walker.node(node.Post)
	walker.stmts(node.Body.List)
}

func (walker scope) switchStmt(node *ast.SwitchStmt) {
	if walker.initShadows(node.Init) {
		return
	}
	walker.node(node.Tag)
	walker.clauses(node.Body)
}

func (walker scope) rangeStmt(node *ast.RangeStmt) {
	walker.node(node.X)
	if node.Tok == token.DEFINE && walker.hasExpr([]ast.Expr{node.Key, node.Value}) {
		return
	}
	walker.stmts(node.Body.List)
}

func (walker scope) typeSwitch(node *ast.TypeSwitchStmt) {
	if walker.initShadows(node.Init) {
		return
	}
	if assign, ok := node.Assign.(*ast.AssignStmt); ok && walker.defines(assign) {
		walker.exprs(assign.Rhs)
		return
	}
	walker.node(node.Assign)
	walker.clauses(node.Body)
}

func (walker scope) clauses(body *ast.BlockStmt) {
	for _, clause := range body.List {
		switch node := clause.(type) {
		case *ast.CaseClause:
			walker.exprs(node.List)
			walker.stmts(node.Body)
		case *ast.CommClause:
			if assign, ok := node.Comm.(*ast.AssignStmt); ok && walker.defines(assign) {
				walker.exprs(assign.Rhs)
				continue
			}
			walker.node(node.Comm)
			walker.stmts(node.Body)
		}
	}
}

func (walker scope) node(root ast.Node) {
	if root == nil {
		return
	}
	ast.Inspect(root, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncLit:
			if !walker.hasField(typed.Type.Params) && !walker.hasField(typed.Type.Results) {
				walker.stmts(typed.Body.List)
			}
			return false
		case *ast.IndexExpr:
			if walker.isTarget(typed.X) {
				walker.report(typed, walker.name)
			}
		}
		return true
	})
}
