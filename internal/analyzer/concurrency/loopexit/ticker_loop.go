package loopexit

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type loopShape struct {
	forPos token.Pos
	body   *ast.BlockStmt
}

func tickerLoopShape(target ast.Node, scope channelScope) (loopShape, bool) {
	switch node := target.(type) {
	case *ast.ForStmt:
		if node.Body == nil || !containsDotCReceive(node.Body, scope) {
			return loopShape{}, false
		}
		return loopShape{forPos: node.For, body: node.Body}, true
	case *ast.RangeStmt:
		if node.Body == nil {
			return loopShape{}, false
		}
		if isRangeTickerPaced(node.X, scope) || containsDotCReceive(node.Body, scope) {
			return loopShape{forPos: node.For, body: node.Body}, true
		}
		return loopShape{}, false
	default:
		return loopShape{}, false
	}
}

func isRangeTickerPaced(expr ast.Expr, scope channelScope) bool {
	if isTimeTickCall(expr) || isTickerDotCSelector(expr, scope) {
		return true
	}
	ident, ok := expr.(*ast.Ident)
	return ok && scope.tickerIdents[ident.Name]
}

func containsDotCReceive(node ast.Node, scope channelScope) bool {
	return containsMatch(node, func(visited ast.Node) bool {
		unary, ok := visited.(*ast.UnaryExpr)
		if !ok || unary.Op != token.ARROW {
			return false
		}
		if isTickerDotCSelector(unary.X, scope) {
			return true
		}
		ident, ok := unary.X.(*ast.Ident)
		return ok && scope.tickerIdents[ident.Name]
	})
}

func containsDoneReceive(node ast.Node, doneIdents map[string]bool) bool {
	return containsMatch(node, func(visited ast.Node) bool {
		return isDoneReceiveNode(visited, doneIdents)
	})
}

func isDoneReceiveNode(node ast.Node, doneIdents map[string]bool) bool {
	exprStmt, ok := node.(*ast.ExprStmt)
	if !ok {
		return false
	}
	unary, ok := exprStmt.X.(*ast.UnaryExpr)
	if !ok || unary.Op != token.ARROW {
		return false
	}
	if isDoneCall(unary.X) {
		return true
	}
	ident, ok := unary.X.(*ast.Ident)
	return ok && doneIdents[ident.Name]
}

func containsMatch(node ast.Node, match func(ast.Node) bool) bool {
	found := false
	ast.Inspect(node, func(visited ast.Node) bool {
		if found {
			return false
		}
		if match(visited) {
			found = true
			return false
		}
		switch visited.(type) {
		case *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit:
			// A nested loop is analysed on its own, and a receive in a nested closure
			// does not pace or stop this one.
			return false
		default:
			return true
		}
	})
	return found
}

func checkTickerLoop(
	check fileCheck, shape loopShape, scope channelScope, label string,
) []pass.Finding {
	if containsDoneReceive(shape.body, scope.doneIdents) || loopCanTerminate(shape.body, label) {
		return nil
	}

	return []pass.Finding{check.reporter.At(
		check.fc.Path,
		check.fset.Position(shape.forPos).Line,
		"ticker-paced loop has no <-ctx.Done() arm; this leaks the goroutine on "+
			"shutdown per concurrency.md #5",
		"add a case <-ctx.Done(): return arm",
	)}
}
