package goast

import (
	"go/ast"
	"go/token"
)

func CyclomaticComplexity(fn *ast.FuncDecl) int {
	score := 1
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CommClause:
			score++
		case *ast.CaseClause:
			if typed.List != nil {
				score++
			}
		case *ast.BinaryExpr:
			if typed.Op == token.LAND || typed.Op == token.LOR {
				score++
			}
		}
		return true
	})
	return score
}

func ParamCount(fields *ast.FieldList) int {
	total := 0
	for _, field := range fields.List {
		total += max(1, len(field.Names))
	}
	return total
}
