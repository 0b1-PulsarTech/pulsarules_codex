package packagevar

import (
	"go/ast"
	"go/token"
	"go/types"
)

type enumScan struct {
	imports  map[string]bool
	consts   map[string]bool
	declared map[string]bool
	vars     map[string]*ast.ValueSpec
}

func enumNames(file *ast.File, imports, consts map[string]bool) map[string]bool {
	scan := enumScan{
		imports:  imports,
		consts:   consts,
		declared: make(map[string]bool),
		vars:     make(map[string]*ast.ValueSpec),
	}
	var groups []*ast.GenDecl
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		scan.collect(gen)
		if len(gen.Specs) >= 2 {
			groups = append(groups, gen)
		}
	}
	exempt := make(map[string]bool)
	for _, gen := range groups {
		scan.exemptGroup(gen, exempt)
	}
	return exempt
}

func (scan enumScan) exemptGroup(gen *ast.GenDecl, exempt map[string]bool) {
	recv, ok := scan.groupRecv(gen)
	if !ok {
		return
	}
	for _, spec := range gen.Specs {
		if vs, isValue := spec.(*ast.ValueSpec); isValue {
			exempt[vs.Names[0].Name] = true
		}
	}
	if scan.isDefinition(scan.vars[recv]) {
		exempt[recv] = true
	}
}

func (scan enumScan) collect(gen *ast.GenDecl) {
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range vs.Names {
			scan.declared[name.Name] = true
		}
		if len(vs.Names) == 1 && len(vs.Values) == 1 {
			scan.vars[vs.Names[0].Name] = vs
		}
	}
}

func (scan enumScan) groupRecv(gen *ast.GenDecl) (recv string, ok bool) {
	var callee string
	for idx, spec := range gen.Specs {
		call, valid := scan.groupCall(spec)
		if !valid {
			return "", false
		}
		text := types.ExprString(call.Fun)
		if idx == 0 {
			callee = text
		}
		if text != callee {
			return "", false
		}
		if idx == 0 {
			recv = scan.calleeRecv(call)
		}
	}
	return recv, true
}

func (scan enumScan) groupCall(spec ast.Spec) (*ast.CallExpr, bool) {
	vs, ok := spec.(*ast.ValueSpec)
	if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 || vs.Names[0].Name == "_" {
		return nil, false
	}
	call, ok := vs.Values[0].(*ast.CallExpr)
	if !ok || !scan.constArgs(call) {
		return nil, false
	}
	_, isImported, isVarRecv := scan.callee(call)
	return call, isImported || isVarRecv
}

func (scan enumScan) callee(call *ast.CallExpr) (name string, isImported, isVarRecv bool) {
	fun := call.Fun
	switch typed := fun.(type) {
	case *ast.IndexExpr:
		fun = typed.X
	case *ast.IndexListExpr:
		fun = typed.X
	}
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return "", false, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false, false
	}
	if scan.imports[ident.Name] {
		return ident.Name, true, false
	}
	return ident.Name, false, scan.declared[ident.Name]
}

func (scan enumScan) calleeRecv(call *ast.CallExpr) string {
	name, _, isVarRecv := scan.callee(call)
	if isVarRecv {
		return name
	}
	return ""
}

func (scan enumScan) isDefinition(vs *ast.ValueSpec) bool {
	if vs == nil {
		return false
	}
	call, ok := vs.Values[0].(*ast.CallExpr)
	if !ok || !scan.constArgs(call) {
		return false
	}
	_, isImported, _ := scan.callee(call)
	return isImported
}

func (scan enumScan) constArgs(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	for _, arg := range call.Args {
		if !scan.isConst(arg) {
			return false
		}
	}
	return true
}

func (scan enumScan) isConst(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return scan.consts[typed.Name]
	case *ast.ParenExpr:
		return scan.isConst(typed.X)
	case *ast.UnaryExpr:
		return scan.isConst(typed.X)
	case *ast.BinaryExpr:
		return scan.isConst(typed.X) && scan.isConst(typed.Y)
	}
	return false
}
