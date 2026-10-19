package goast

import (
	"go/ast"
	"go/token"
)

type ResultSite struct {
	Name    string
	Pos     token.Pos
	Results *ast.FieldList
}

func FuncDeclSite(fn *ast.FuncDecl) ResultSite {
	return ResultSite{Name: fn.Name.Name, Pos: fn.Pos(), Results: fn.Type.Results}
}

func InterfaceMethodSites(iface *ast.InterfaceType) []ResultSite {
	var sites []ResultSite
	for _, field := range iface.Methods.List {
		funcType, ok := field.Type.(*ast.FuncType)
		// An embedded interface has no method name of its own to report.
		if !ok || len(field.Names) == 0 {
			continue
		}
		sites = append(sites, ResultSite{
			Name: field.Names[0].Name, Pos: field.Pos(), Results: funcType.Results,
		})
	}
	return sites
}
