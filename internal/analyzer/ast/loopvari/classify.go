package loopvari

import (
	"go/ast"
	"go/token"
	"slices"
)

func (s *scanner) classify(ident *ast.Ident) bool {
	if s.isNameOnly(ident) {
		return false
	}
	if s.closure > 0 {
		return true
	}
	cur := ast.Node(ident)
	depth := len(s.stack) - 1
	for depth >= 0 && isTransparent(s.stack[depth]) {
		cur = s.stack[depth]
		depth--
	}
	if depth < 0 {
		return true
	}
	switch parent := s.stack[depth].(type) {
	case *ast.SelectorExpr:
		return parent.X == cur
	case *ast.IndexExpr:
		return parent.Index != cur
	case *ast.SliceExpr:
		return parent.X == cur
	case *ast.BinaryExpr:
		return !s.isBoundCheck(depth, cur)
	case *ast.AssignStmt:
		return !s.isInnerLoopStart(depth, cur)
	}
	return true
}

func (s *scanner) isNameOnly(ident *ast.Ident) bool {
	if len(s.stack) == 0 {
		return false
	}
	switch parent := s.stack[len(s.stack)-1].(type) {
	case *ast.LabeledStmt, *ast.BranchStmt:
		return true
	case *ast.KeyValueExpr:
		return parent.Key == ast.Expr(ident) && !s.keysAreValues(len(s.stack)-2)
	}
	return false
}

func isTransparent(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.ParenExpr:
		return true
	case *ast.BinaryExpr:
		return node.Op == token.ADD || node.Op == token.SUB
	case *ast.CallExpr:
		return isIntegerConversion(node)
	}
	return false
}

func isIntegerConversion(call *ast.CallExpr) bool {
	fn, ok := call.Fun.(*ast.Ident)
	if !ok || len(call.Args) != 1 {
		return false
	}
	switch fn.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	}
	return false
}

func (s *scanner) isInnerLoopStart(depth int, operand ast.Node) bool {
	assign, ok := s.stack[depth].(*ast.AssignStmt)
	if !ok || depth == 0 || (assign.Tok != token.DEFINE && assign.Tok != token.ASSIGN) {
		return false
	}
	if !slices.ContainsFunc(
		assign.Rhs,
		func(rhs ast.Expr) bool { return ast.Node(rhs) == operand },
	) {
		return false
	}
	loop, ok := s.stack[depth-1].(*ast.ForStmt)
	return ok && loop.Init == ast.Stmt(assign)
}

func (s *scanner) isBoundCheck(depth int, operand ast.Node) bool {
	cmp, ok := s.stack[depth].(*ast.BinaryExpr)
	if !ok {
		return false
	}
	switch cmp.Op {
	case token.LSS, token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ:
	default:
		return false
	}
	other := cmp.X
	if ast.Node(cmp.X) == operand {
		other = cmp.Y
	}
	if s.inCond || s.isInnerLoopCond(depth, other) {
		return true
	}
	return s.isLenBound(other)
}

func (s *scanner) isInnerLoopCond(depth int, other ast.Expr) bool {
	child := s.stack[depth]
	for pos := depth - 1; pos >= 0; pos-- {
		switch parent := s.stack[pos].(type) {
		case *ast.ForStmt:
			return parent.Cond == child && countsInner(parent, other)
		case *ast.ParenExpr:
		case *ast.BinaryExpr:
			if parent.Op != token.LAND && parent.Op != token.LOR {
				return false
			}
		default:
			return false
		}
		child = s.stack[pos]
	}
	return false
}
