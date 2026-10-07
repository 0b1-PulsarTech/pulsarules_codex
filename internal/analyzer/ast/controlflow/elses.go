package controlflow

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (analyzer *Analyzer) checkElseBlock(
	block *ast.BlockStmt,
	emit func(token.Pos, string, string),
) {
	for _, stmt := range block.List {
		switch typed := stmt.(type) {
		case *ast.ReturnStmt:
			emit(
				typed.Return,
				"else block contains return; use early return instead",
				"remove else and return directly",
			)
			return
		case *ast.ExprStmt:
			if call, ok := typed.X.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
					emit(
						typed.Pos(),
						"else block contains panic; panic directly instead",
						"remove else and panic directly",
					)
					return
				}
			}
		case *ast.ForStmt, *ast.RangeStmt:
			emit(
				typed.Pos(),
				"else block contains loop; restructure to avoid else",
				"move loop out of else or invert the condition",
			)
			return
		case *ast.BranchStmt:
			emit(
				typed.Pos(),
				fmt.Sprintf("else block contains %s statement", typed.Tok),
				"restructure to avoid branch inside else",
			)
			return
		}
	}
}
