package pproflabel

import (
	"go/ast"
	"go/token"
	"strconv"
)

func pprofImportName(file *ast.File) (string, bool) {
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "runtime/pprof" {
			continue
		}
		if imp.Name == nil {
			return "pprof", true
		}
		if imp.Name.Name == "_" {
			return "", false
		}
		return imp.Name.Name, true
	}
	return "", false
}

func funcSignature(owner ast.Node) (*ast.FieldList, *ast.BlockStmt) {
	switch node := owner.(type) {
	case *ast.FuncDecl:
		return node.Type.Params, node.Body
	case *ast.FuncLit:
		return node.Type.Params, node.Body
	default:
		return nil, nil
	}
}

func isNameShadowed(params *ast.FieldList, body *ast.BlockStmt, name string) bool {
	if paramShadows(params, name) {
		return true
	}
	shadowed := false
	ast.Inspect(body, func(node ast.Node) bool {
		if shadowed {
			return false
		}
		// A literal is a different function, scoped by its own call.
		if _, isFuncLit := node.(*ast.FuncLit); isFuncLit {
			return false
		}
		if declaresName(node, name) {
			shadowed = true
			return false
		}
		return true
	})
	return shadowed
}

func paramShadows(params *ast.FieldList, name string) bool {
	if params == nil {
		return false
	}
	for _, field := range params.List {
		for _, ident := range field.Names {
			if ident.Name == name {
				return true
			}
		}
	}
	return false
}

func declaresName(candidate ast.Node, name string) bool {
	switch node := candidate.(type) {
	case *ast.AssignStmt:
		return node.Tok == token.DEFINE && exprListHasIdent(node.Lhs, name)
	case *ast.ValueSpec:
		return identListHasName(node.Names, name)
	default:
		return false
	}
}

func exprListHasIdent(exprs []ast.Expr, name string) bool {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

func identListHasName(idents []*ast.Ident, name string) bool {
	for _, ident := range idents {
		if ident.Name == name {
			return true
		}
	}
	return false
}
