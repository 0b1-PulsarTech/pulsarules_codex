package loopvari

import (
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

func markBuiltin(shadowed map[string]bool, idents ...*ast.Ident) {
	for _, ident := range idents {
		if ident.Name == "len" || ident.Name == "cap" {
			shadowed[ident.Name] = true
		}
	}
}

func packageShadows(f *ast.File) map[string]bool {
	shadowed := map[string]bool{}
	for _, decl := range f.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if node.Recv == nil {
				markBuiltin(shadowed, node.Name)
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				markSpec(shadowed, spec)
			}
		}
	}
	return shadowed
}

func markSpec(shadowed map[string]bool, spec ast.Spec) {
	switch node := spec.(type) {
	case *ast.ValueSpec:
		markBuiltin(shadowed, node.Names...)
	case *ast.TypeSpec:
		markBuiltin(shadowed, node.Name)
	}
}

func declShadows(pkgLevel map[string]bool, decl ast.Decl, loop ast.Node) map[string]bool {
	shadowed := maps.Clone(pkgLevel)
	var stack []ast.Node
	ast.Inspect(decl, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if idents := declared(n); len(idents) > 0 && visibleAt(stack, n, loop) {
			markBuiltin(shadowed, idents...)
		}
		stack = append(stack, n)
		return true
	})
	return shadowed
}

func declared(n ast.Node) []*ast.Ident {
	switch node := n.(type) {
	case *ast.ValueSpec:
		return node.Names
	case *ast.TypeSpec:
		return []*ast.Ident{node.Name}
	case *ast.FuncDecl:
		return funcFields(node.Type, node.Recv)
	case *ast.FuncLit:
		return funcFields(node.Type, nil)
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			return identsOf(node.Lhs...)
		}
	case *ast.RangeStmt:
		if node.Tok == token.DEFINE {
			return identsOf(node.Key, node.Value)
		}
	}
	return nil
}

func funcFields(typ *ast.FuncType, recv *ast.FieldList) []*ast.Ident {
	var idents []*ast.Ident
	for _, list := range []*ast.FieldList{recv, typ.TypeParams, typ.Params, typ.Results} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			idents = append(idents, field.Names...)
		}
	}
	return idents
}

func identsOf(exprs ...ast.Expr) []*ast.Ident {
	var idents []*ast.Ident
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok {
			idents = append(idents, ident)
		}
	}
	return idents
}

func visibleAt(stack []ast.Node, n, loop ast.Node) bool {
	var scope ast.Node
	switch node := n.(type) {
	case *ast.FuncDecl, *ast.FuncLit:
		scope = n
	case *ast.RangeStmt:
		scope = node.Body
	default:
		if n.End() > loop.Pos() {
			return false
		}
		scope = innermostScope(stack)
	}
	return scope != nil && scope.Pos() <= loop.Pos() && loop.End() <= scope.End()
}

func innermostScope(stack []ast.Node) ast.Node {
	for _, node := range slices.Backward(stack) {
		switch node.(type) {
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause, *ast.ForStmt,
			*ast.RangeStmt, *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			return node
		}
	}
	return nil
}
