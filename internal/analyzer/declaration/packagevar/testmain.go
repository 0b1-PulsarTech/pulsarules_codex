package packagevar

import (
	"go/ast"
	"go/token"
)

type assignTracker struct {
	declared map[string]bool
	assigned map[string]bool
}

func collectAssigned(body *ast.BlockStmt, into map[string]bool) {
	tracker := assignTracker{declared: make(map[string]bool), assigned: make(map[string]bool)}
	ast.Inspect(body, tracker.visit)
	for name := range tracker.assigned {
		if !tracker.declared[name] {
			into[name] = true
		}
	}
}

func (tracker assignTracker) visit(node ast.Node) bool {
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		tracker.trackAssign(stmt)
	case *ast.IncDecStmt:
		markRoot(tracker.assigned, stmt.X)
	case *ast.UnaryExpr:
		if stmt.Op == token.AND {
			markRoot(tracker.assigned, stmt.X)
		}
	case *ast.ValueSpec:
		for _, name := range stmt.Names {
			tracker.declared[name.Name] = true
		}
	case *ast.RangeStmt:
		if stmt.Tok == token.DEFINE {
			markRoot(tracker.declared, stmt.Key)
			markRoot(tracker.declared, stmt.Value)
		}
	}
	return true
}

func (tracker assignTracker) trackAssign(stmt *ast.AssignStmt) {
	into := tracker.assigned
	if stmt.Tok == token.DEFINE {
		into = tracker.declared
	}
	for _, lhs := range stmt.Lhs {
		markRoot(into, lhs)
	}
}

func markRoot(into map[string]bool, expr ast.Expr) {
	for expr != nil {
		switch node := expr.(type) {
		case *ast.Ident:
			into[node.Name] = true
			return
		case *ast.SelectorExpr:
			expr = node.X
		case *ast.IndexExpr:
			expr = node.X
		case *ast.StarExpr:
			expr = node.X
		case *ast.ParenExpr:
			expr = node.X
		default:
			return
		}
	}
}

func isTestMain(fn *ast.FuncDecl) bool {
	return fn.Recv == nil && fn.Name.Name == "TestMain" && fn.Body != nil
}
