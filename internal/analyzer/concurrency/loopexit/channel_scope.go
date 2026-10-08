package loopexit

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

type channelScope struct {
	file             *ast.File
	tickerIdents     map[string]bool
	doneIdents       map[string]bool
	typedIdents      map[string]bool
	tickerFieldNames map[string]bool
}

func isTimeTickCall(file *ast.File, expr ast.Expr) bool {
	return isTimeCall(file, expr, "Tick")
}

func isTimeNewTickerCall(file *ast.File, expr ast.Expr) bool {
	return isTimeCall(file, expr, "NewTicker")
}

func isTimeCall(file *ast.File, expr ast.Expr, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	return ok && goast.PkgCall(file, call, "time", name)
}

func isDotCSelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel != nil && sel.Sel.Name == "C"
}

func isTickerDotCSelector(expr ast.Expr, scope channelScope) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || !isDotCSelector(expr) {
		return false
	}
	if isTimeNewTickerCall(scope.file, sel.X) {
		return true
	}
	switch base := sel.X.(type) {
	case *ast.Ident:
		return scope.tickerIdents[base.Name] || scope.typedIdents[base.Name]
	case *ast.SelectorExpr:
		return base.Sel != nil && scope.tickerFieldNames[base.Sel.Name]
	default:
		return false
	}
}

func isTickerOrTimerType(file *ast.File, expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	return goast.PkgRef(file, expr, "time", "Ticker", "Timer")
}

func isDoneCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel != nil && sel.Sel.Name == "Done"
}

func collectChannelScope(
	file *ast.File, params *ast.FieldList, body *ast.BlockStmt, tickerFieldNames map[string]bool,
) channelScope {
	scope := channelScope{
		file:             file,
		tickerIdents:     map[string]bool{},
		doneIdents:       map[string]bool{},
		typedIdents:      map[string]bool{},
		tickerFieldNames: tickerFieldNames,
	}
	recordTypedParams(params, scope)
	ast.Inspect(body, func(node ast.Node) bool {
		// A literal is a different function, scoped by its own call.
		if _, isFuncLit := node.(*ast.FuncLit); isFuncLit {
			return false
		}
		switch typed := node.(type) {
		case *ast.AssignStmt:
			recordChannelAssign(typed, scope)
		case *ast.ValueSpec:
			recordTypedVar(typed, scope)
		}
		return true
	})
	return scope
}

func recordTypedParams(params *ast.FieldList, scope channelScope) {
	if params == nil {
		return
	}
	for _, field := range params.List {
		if !isTickerOrTimerType(scope.file, field.Type) {
			continue
		}
		for _, ident := range field.Names {
			scope.typedIdents[ident.Name] = true
		}
	}
}

func recordTypedVar(spec *ast.ValueSpec, scope channelScope) {
	if spec.Type == nil || !isTickerOrTimerType(scope.file, spec.Type) {
		return
	}
	for _, ident := range spec.Names {
		scope.typedIdents[ident.Name] = true
	}
}

func recordChannelAssign(assign *ast.AssignStmt, scope channelScope) {
	for index, rhs := range assign.Rhs {
		if index >= len(assign.Lhs) {
			return
		}
		ident, ok := assign.Lhs[index].(*ast.Ident)
		if !ok {
			continue
		}
		switch {
		case isTimeTickCall(scope.file, rhs) || isTimeNewTickerCall(scope.file, rhs):
			scope.tickerIdents[ident.Name] = true
		case isDoneCall(rhs):
			scope.doneIdents[ident.Name] = true
		}
	}
}
