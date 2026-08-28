package contextdiscipline

import (
	"go/ast"
	"go/token"
)

type breakScope int

const (
	inLoop breakScope = iota
	absorbed
)

func loopCanTerminate(body *ast.BlockStmt, label string) bool {
	hasReturn := containsMatch(body, func(node ast.Node) bool {
		_, ok := node.(*ast.ReturnStmt)
		return ok
	})
	return hasReturn || hasLoopExitingBreak(body, label)
}

func hasLoopExitingBreak(body *ast.BlockStmt, label string) bool {
	return searchLoopExitingBreak(body, label, inLoop)
}

func searchLoopExitingBreak(node ast.Node, label string, scope breakScope) bool {
	found := false
	// A break inside a nested for/switch/select belongs to it, unless labeled.
	ast.Inspect(node, func(visited ast.Node) bool {
		if found || visited == nil {
			return false
		}
		switch stmt := visited.(type) {
		case *ast.BranchStmt:
			if stmt.Tok == token.BREAK {
				found = breakExitsLoop(stmt, label, scope)
			}
			return false
		case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			found = searchLoopExitingBreak(nestedBody(visited), label, absorbed)
			return false
		case *ast.FuncLit:
			return false
		default:
			return true
		}
	})
	return found
}

func nestedBody(node ast.Node) *ast.BlockStmt {
	switch stmt := node.(type) {
	case *ast.ForStmt:
		return stmt.Body
	case *ast.RangeStmt:
		return stmt.Body
	case *ast.SwitchStmt:
		return stmt.Body
	case *ast.TypeSwitchStmt:
		return stmt.Body
	case *ast.SelectStmt:
		return stmt.Body
	}
	return nil
}

func breakExitsLoop(stmt *ast.BranchStmt, label string, scope breakScope) bool {
	if stmt.Label == nil {
		switch scope {
		case inLoop:
			return true
		case absorbed:
			return false
		}
	}
	return label != "" && stmt.Label.Name == label
}

func collectLoopLabels(body *ast.BlockStmt) map[ast.Stmt]string {
	labels := map[ast.Stmt]string{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, isFuncLit := node.(*ast.FuncLit); isFuncLit {
			return false
		}
		labeled, ok := node.(*ast.LabeledStmt)
		if !ok {
			return true
		}
		switch labeled.Stmt.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			labels[labeled.Stmt] = labeled.Label.Name
		}
		return true
	})
	return labels
}
