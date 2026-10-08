package tsfacts

import "github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

func isTestCallback(node *ast.Node) bool {
	call := node.Parent
	if call == nil || call.Kind != ast.KindCallExpression {
		return false
	}
	isArgument := false
	for _, argument := range call.Arguments() {
		isArgument = isArgument || argument == node
	}
	return isArgument && isTestCallee(call.AsCallExpression().Expression)
}

func isTestCallee(callee *ast.Node) bool {
	if callee.Kind == ast.KindPropertyAccessExpression {
		callee = callee.Expression()
	}
	if callee.Kind != ast.KindIdentifier {
		return false
	}
	switch callee.Text() {
	case "describe", "it", "test", "suite", "specify", "beforeEach", "afterEach",
		"beforeAll", "afterAll", "before", "after":
		return true
	}
	return false
}
