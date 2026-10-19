package goast

import (
	"go/ast"
	"go/token"
)

type FuncSite struct {
	Name    string
	Pos     token.Pos
	Results *ast.FieldList
	Body    *ast.BlockStmt
}

func FuncDeclSite(fn *ast.FuncDecl) FuncSite {
	return FuncSite{Name: fn.Name.Name, Pos: fn.Pos(), Results: fn.Type.Results, Body: fn.Body}
}

func FuncLitSite(lit *ast.FuncLit) FuncSite {
	return FuncSite{
		Name: "function literal", Pos: lit.Pos(), Results: lit.Type.Results, Body: lit.Body,
	}
}

func InterfaceMethodSites(iface *ast.InterfaceType) []FuncSite {
	var sites []FuncSite
	for _, field := range iface.Methods.List {
		funcType, ok := field.Type.(*ast.FuncType)
		// An embedded interface has no method name of its own to report.
		if !ok || len(field.Names) == 0 {
			continue
		}
		sites = append(sites, FuncSite{
			Name: field.Names[0].Name, Pos: field.Pos(), Results: funcType.Results,
		})
	}
	return sites
}
