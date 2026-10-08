package registry

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"slices"
)

func (pkg *typedPackage) isParamSet(expr ast.Expr) bool {
	typed := pkg.info.Types[expr]
	return !typed.IsType() && isParamSetType(typed.Type)
}

func (pkg *typedPackage) isConversion(expr ast.Expr) bool {
	call, isCall := ast.Unparen(expr).(*ast.CallExpr)
	return isCall && pkg.info.Types[call.Fun].IsType()
}

func (pkg *typedPackage) rawConfigField(node ast.Node) string {
	sel, isSel := node.(*ast.SelectorExpr)
	selection := pkg.info.Selections[sel]
	if !isSel || selection == nil || selection.Kind() != types.FieldVal {
		return ""
	}
	owner, isNamed := types.Unalias(deref(selection.Recv())).(*types.Named)
	if !isNamed || owner.Obj().Pkg() == nil || owner.Obj().Pkg().Path() != analyzerPath+"/pass" {
		return ""
	}
	name := owner.Obj().Name() + "." + selection.Obj().Name()
	if name == "AnalysisConfig.Analyzers" || name == "AnalyzerConfig.Params" {
		return name
	}
	return ""
}

func deref(typ types.Type) types.Type {
	if pointer, isPointer := typ.(*types.Pointer); isPointer {
		return pointer.Elem()
	}
	return typ
}

func (pkg *typedPackage) foreignParams(receiver ast.Expr, fn *ast.FuncDecl) error {
	values := []ast.Expr{ast.Unparen(receiver)}
	if ident, isIdent := values[0].(*ast.Ident); isIdent {
		target := pkg.info.Uses[ident]
		if target == nil || boundOutside(target, ident, fn) {
			return nil
		}
		values = pkg.assignedValues(target, fn)
	}
	for _, value := range values {
		call, isCall := ast.Unparen(value).(*ast.CallExpr)
		if !isCall || pkg.isConversion(call) || !pkg.isParamSet(call) ||
			coreName(pkg.info.Uses[calleeIdent(call)]) == "Context.Params" {
			continue
		}
		return fmt.Errorf("params come from %s, not Context.Params",
			types.ExprString(call))
	}
	return nil
}

func (pkg *typedPackage) scanParamsCall(site callSite, out *scanResult) {
	if site.call == nil {
		out.add(pkg.dir, nil, nil, errors.New("Params is used as a value, not called"))
		return
	}
	sel, _ := ast.Unparen(site.call.Fun).(*ast.SelectorExpr)
	selection := pkg.info.Selections[sel]
	if selection == nil || selection.Kind() != types.MethodVal {
		out.add(pkg.dir, nil, nil, errors.New("a method expression reads Params"))
		return
	}
	if !isContextType(pkg.info.Types[sel.X].Type) {
		out.add(pkg.dir, nil, nil, fmt.Errorf("Params is called on %s, not a Context",
			types.ExprString(sel.X)))
	}
}

func isContextType(typ types.Type) bool {
	named, isNamed := types.Unalias(deref(typ)).(*types.Named)
	return isNamed && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == analyzerPath+"/pass"
}

func (pkg *typedPackage) attributedParamsCalls() map[*ast.CallExpr]bool {
	attributed := map[*ast.CallExpr]bool{}
	for ident, site := range pkg.callees {
		sel, isSel := ast.Unparen(site.call.Fun).(*ast.SelectorExpr)
		if !isSel || !isReader(coreName(pkg.info.Uses[ident])) {
			continue
		}
		if call := pkg.paramsCall(sel.X, site.fn); call != nil {
			attributed[call] = true
		}
	}
	return attributed
}

func (pkg *typedPackage) scanOtherParamIDs(result *scanResult, dirIDs []string) {
	attributed := pkg.attributedParamsCalls()
	for _, file := range pkg.files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, isCall := node.(*ast.CallExpr)
			if !isCall || coreName(pkg.info.Uses[calleeIdent(call)]) != "Context.Params" {
				return true
			}
			ids, err := pkg.resolve(call.Args[0])
			if err != nil && !attributed[call] {
				result.add(pkg.dir, nil, nil, err)
				return true
			}
			if err == nil && !attributed[call] && !slices.Contains(dirIDs, ids[0]) {
				result.add(pkg.dir, nil, nil, fmt.Errorf(
					"%s names an analyzer outside %v and no read is attributed to it",
					types.ExprString(call), dirIDs))
			}
			return true
		})
	}
}

func (pkg *typedPackage) scanParamUses(result *scanResult, dirIDs []string) {
	pkg.scanOtherParamIDs(result, dirIDs)
	for _, file := range pkg.files {
		allowed := pkg.allowedUses(file, dirIDs)
		ast.Inspect(file, func(node ast.Node) bool {
			expr, isExpr := node.(ast.Expr)
			if isExpr && pkg.isParamSet(expr) && !allowed[expr] {
				err := fmt.Errorf("params are used outside a reader call: %s",
					types.ExprString(expr))
				result.add(pkg.dir, nil, nil, err)
			}
			if field := pkg.rawConfigField(node); field != "" {
				err := fmt.Errorf("%s is read directly, bypassing the reader calls", field)
				result.add(pkg.dir, nil, nil, err)
			}
			return true
		})
	}
}

func (pkg *typedPackage) allowedUses(file *ast.File, dirIDs []string) map[ast.Expr]bool {
	allowed := map[ast.Expr]bool{}
	allow := func(expr ast.Expr) {
		for {
			allowed[expr] = true
			paren, isParen := expr.(*ast.ParenExpr)
			if !isParen {
				return
			}
			expr = paren.X
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		pkg.allowAt(node, dirIDs, allow)
		return true
	})
	return allowed
}

func (pkg *typedPackage) allowAt(node ast.Node, dirIDs []string, allow func(ast.Expr)) {
	switch stmt := node.(type) {
	case *ast.SelectorExpr:
		if isReader(coreName(pkg.info.Uses[stmt.Sel])) && !pkg.isConversion(stmt.X) {
			allow(stmt.X)
		}
	case *ast.CallExpr:
		pkg.allowArguments(stmt, dirIDs, allow)
	case *ast.AssignStmt:
		pkg.allowAssigned(identsOf(stmt.Lhs), stmt.Rhs, allow)
	case *ast.ValueSpec:
		pkg.allowAssigned(stmt.Names, stmt.Values, allow)
	case *ast.RangeStmt:
		for _, target := range []ast.Expr{stmt.Key, stmt.Value} {
			if target != nil {
				allow(target)
			}
		}
	}
}

func (pkg *typedPackage) allowAssigned(
	names []*ast.Ident, values []ast.Expr, allow func(ast.Expr),
) {
	if len(names) != len(values) {
		return
	}
	for idx, name := range names {
		if name == nil {
			continue
		}
		obj := pkg.info.Defs[name]
		if obj == nil {
			obj = pkg.info.Uses[name]
		}
		if isLocalVar(obj) && isParamSetType(obj.Type()) && !pkg.isConversion(values[idx]) {
			allow(name)
			allow(values[idx])
		}
	}
}

func isLocalVar(obj types.Object) bool {
	variable, isVar := obj.(*types.Var)
	return isVar && variable.Parent() != variable.Pkg().Scope()
}

func (pkg *typedPackage) allowArguments(
	call *ast.CallExpr, dirIDs []string, allow func(ast.Expr),
) {
	callee, isFunc := pkg.info.Uses[calleeIdent(call)].(*types.Func)
	if !isFunc {
		return
	}
	callee = callee.Origin()
	if pkg.decls[callee] == nil {
		return
	}
	sig := callee.Signature()
	fixed := sig.Params().Len()
	if sig.Variadic() {
		fixed--
	}
	for idx, arg := range call.Args {
		if idx < fixed && isParamSetType(sig.Params().At(idx).Type()) &&
			pkg.passesOwnParams(arg, call, dirIDs) {
			allow(arg)
		}
	}
}

func (pkg *typedPackage) passesOwnParams(arg ast.Expr, call *ast.CallExpr, dirIDs []string) bool {
	site, found := pkg.callees[calleeIdent(call)]
	if !found {
		return false
	}
	source := pkg.paramsCall(arg, site.fn)
	if source == nil {
		return false
	}
	ids, err := pkg.resolve(source.Args[0])
	return err == nil && slices.Equal(ids, dirIDs)
}
