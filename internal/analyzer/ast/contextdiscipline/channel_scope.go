package contextdiscipline

import "go/ast"

type channelScope struct {
	tickerIdents     map[string]bool
	doneIdents       map[string]bool
	typedIdents      map[string]bool
	tickerFieldNames map[string]bool
}

func isTimeTickCall(expr ast.Expr) bool {
	return isTimePkgCall(expr, "Tick")
}

func isTimeNewTickerCall(expr ast.Expr) bool {
	return isTimePkgCall(expr, "NewTicker")
}

func isTimePkgCall(expr ast.Expr, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != name {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
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
	if isTimeNewTickerCall(sel.X) {
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

func isTickerOrTimerType(expr ast.Expr) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time" && (sel.Sel.Name == "Ticker" || sel.Sel.Name == "Timer")
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
	params *ast.FieldList, body *ast.BlockStmt, tickerFieldNames map[string]bool,
) channelScope {
	scope := channelScope{
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
		if !isTickerOrTimerType(field.Type) {
			continue
		}
		for _, ident := range field.Names {
			scope.typedIdents[ident.Name] = true
		}
	}
}

func recordTypedVar(spec *ast.ValueSpec, scope channelScope) {
	if spec.Type == nil || !isTickerOrTimerType(spec.Type) {
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
		case isTimeTickCall(rhs) || isTimeNewTickerCall(rhs):
			scope.tickerIdents[ident.Name] = true
		case isDoneCall(rhs):
			scope.doneIdents[ident.Name] = true
		}
	}
}
