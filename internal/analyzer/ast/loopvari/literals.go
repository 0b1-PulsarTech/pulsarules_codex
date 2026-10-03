package loopvari

import (
	"go/ast"
	"slices"
)

func (s *scanner) keysAreValues(depth int) bool {
	if depth < 0 {
		return false
	}
	lit, ok := s.stack[depth].(*ast.CompositeLit)
	if !ok {
		return false
	}
	if lit.Type != nil {
		_, isMap := lit.Type.(*ast.MapType)
		return isMap
	}
	var inKeys []bool
	inKey := false
	child := ast.Node(lit)
	for depth--; depth >= 0; depth-- {
		switch outer := s.stack[depth].(type) {
		case *ast.KeyValueExpr:
			inKey = ast.Node(outer.Key) == child
		case *ast.CompositeLit:
			inKeys = append(inKeys, inKey)
			inKey = false
			if outer.Type != nil {
				return holdsMaps(outer.Type, inKeys)
			}
		}
		child = s.stack[depth]
	}
	return false
}

func holdsMaps(typ ast.Expr, inKeys []bool) bool {
	for _, inKey := range slices.Backward(inKeys) {
		switch node := unstar(typ).(type) {
		case *ast.ArrayType:
			typ = node.Elt
		case *ast.MapType:
			typ = node.Value
			if inKey {
				typ = node.Key
			}
		default:
			return false
		}
	}
	_, isMap := unstar(typ).(*ast.MapType)
	return isMap
}

func unstar(typ ast.Expr) ast.Expr {
	for {
		switch node := typ.(type) {
		case *ast.StarExpr:
			typ = node.X
		case *ast.ParenExpr:
			typ = node.X
		default:
			return typ
		}
	}
}
