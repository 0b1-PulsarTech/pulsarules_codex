package contextdiscipline

import "go/ast"

func collectTickerFieldNames(file *ast.File) map[string]bool {
	fieldNames := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		structType, ok := node.(*ast.StructType)
		if !ok || structType.Fields == nil {
			return true
		}
		for _, field := range structType.Fields.List {
			if !isTickerOrTimerType(field.Type) {
				continue
			}
			for _, ident := range field.Names {
				fieldNames[ident.Name] = true
			}
		}
		return true
	})
	return fieldNames
}
