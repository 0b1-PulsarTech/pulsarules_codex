package goast

import "go/ast"

func FieldCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	total := 0
	for _, field := range fields.List {
		total += max(1, len(field.Names))
	}
	return total
}
