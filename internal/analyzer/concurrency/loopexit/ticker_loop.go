package loopexit

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
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
	if isTimeTickCall(scope.file, expr) || isTickerDotCSelector(expr, scope) {
		return true
	}
	ident, ok := expr.(*ast.Ident)
	return ok && scope.tickerIdents[ident.Name]
}

func containsDotCReceive(node ast.Node, scope channelScope) bool {
	return goast.AnyUntil(node, func(visited ast.Node) bool {
		unary, ok := visited.(*ast.UnaryExpr)
		if !ok || unary.Op != token.ARROW {
			return false
		}
		if isTickerDotCSelector(unary.X, scope) {
			return true
		}
		ident, ok := unary.X.(*ast.Ident)
		return ok && scope.tickerIdents[ident.Name]
	}, isNestedLoopOrFunc)
}

func containsDoneReceive(node ast.Node, doneIdents map[string]bool) bool {
	return goast.AnyUntil(node, func(visited ast.Node) bool {
		return isDoneReceiveNode(visited, doneIdents)
	}, isNestedLoopOrFunc)
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

func isNestedLoopOrFunc(node ast.Node) bool {
	// A nested loop is analysed on its own; a nested closure's receive paces neither.
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit:
		return true
	default:
		return false
	}
}

func checkTickerLoop(
	check fileCheck, shape loopShape, scope channelScope, label string,
) []finding.Finding {
	if containsDoneReceive(shape.body, scope.doneIdents) || loopCanTerminate(shape.body, label) {
		return nil
	}

	forLine := check.fset.Position(shape.forPos).Line
	reason, hasMarker := findProcessLifetimeMarker(
		check.fset,
		check.comments,
		forLine,
		check.codeLines,
	)
	if hasMarker && reason != "" {
		return nil
	}
	if hasMarker {
		return []finding.Finding{check.reporter.At(
			check.fc.Path,
			forLine,
			"pulsarules:process-lifetime marker has no stated reason; explain "+
				"why this loop is deliberately process-scoped",
			"add a reason after the marker, e.g. //pulsarules:process-lifetime <reason>",
		)}
	}
	return []finding.Finding{check.reporter.At(
		check.fc.Path,
		forLine,
		"ticker-paced loop has no <-ctx.Done() arm and no "+
			"//pulsarules:process-lifetime marker; this leaks the goroutine on "+
			"shutdown per concurrency.md #5",
		"add a case <-ctx.Done(): return arm, or mark the loop "+
			"//pulsarules:process-lifetime <reason> if it truly lives for the process",
	)}
}
