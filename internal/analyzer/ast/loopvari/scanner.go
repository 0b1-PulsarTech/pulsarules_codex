package loopvari

import "go/ast"

type scanner struct {
	stack    []ast.Node
	shadowed map[string]bool
	inCond   bool
	closure  int
	escaped  bool
}

func newScanner(shadowed map[string]bool) *scanner {
	return &scanner{shadowed: shadowed}
}

func (s *scanner) scanNode(root ast.Node) bool {
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			s.pop()
			return true
		}
		if s.escaped || !s.descend(n) {
			return false
		}
		s.stack = append(s.stack, n)
		if _, ok := n.(*ast.FuncLit); ok {
			s.closure++
		}
		return true
	})
	return s.escaped
}

func (s *scanner) pop() {
	top := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	if _, ok := top.(*ast.FuncLit); ok {
		s.closure--
	}
}

func (s *scanner) descend(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.BlockStmt:
		s.block(node, node.List)
		return false
	case *ast.CaseClause:
		s.scanUnder(node, exprNodes(node.List)...)
		s.block(node, node.Body)
		return false
	case *ast.CommClause:
		s.commClause(node)
		return false
	case *ast.FuncLit:
		return !fieldsNameLoopVar(node.Type.Params) && !fieldsNameLoopVar(node.Type.Results)
	case *ast.Ident:
		if node.Name == loopVarName && s.classify(node) {
			s.escaped = true
		}
		return true
	}
	return !s.shadows(n)
}

func (s *scanner) shadows(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.IfStmt:
		return s.shadowInit(node, node.Init)
	case *ast.SwitchStmt:
		return s.shadowInit(node, node.Init)
	case *ast.TypeSwitchStmt:
		return s.shadowInit(node, node.Init) || s.shadowInit(node, node.Assign)
	case *ast.ForStmt:
		return s.shadowInit(node, node.Init)
	case *ast.RangeStmt:
		return s.shadowRange(node)
	}
	return false
}

func (s *scanner) commClause(clause *ast.CommClause) {
	if assign := definedAssign(clause.Comm); assign != nil {
		s.scanUnder(clause, exprNodes(assign.Rhs)...)
		return
	}
	if clause.Comm != nil {
		s.scanUnder(clause, clause.Comm)
	}
	s.block(clause, clause.Body)
}

func (s *scanner) shadowInit(owner ast.Node, init ast.Stmt) bool {
	assign := definedAssign(init)
	if assign == nil {
		return false
	}
	s.scanUnder(owner, exprNodes(assign.Rhs)...)
	return true
}

func (s *scanner) shadowRange(loop *ast.RangeStmt) bool {
	if !definesLoopVar(loop.Key, loop.Tok) && !definesLoopVar(loop.Value, loop.Tok) {
		return false
	}
	s.scanUnder(loop, loop.X)
	return true
}

func (s *scanner) scanUnder(owner ast.Node, nodes ...ast.Node) {
	s.stack = append(s.stack, owner)
	for _, node := range nodes {
		s.scanNode(node)
	}
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *scanner) block(owner ast.Node, stmts []ast.Stmt) {
	s.stack = append(s.stack, owner)
	for _, stmt := range stmts {
		if s.escaped {
			break
		}
		if assign := definedAssign(stmt); assign != nil {
			s.scanUnder(assign, exprNodes(assign.Rhs)...)
			break
		}
		if gen := loopVarDecl(stmt); gen != nil {
			s.scanDeclValues(gen)
			break
		}
		s.scanNode(stmt)
	}
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *scanner) scanDeclValues(gen *ast.GenDecl) {
	for _, spec := range gen.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		s.scanUnder(valueSpec, exprNodes(valueSpec.Values)...)
		if namesLoopVar(valueSpec.Names) {
			return
		}
	}
}
