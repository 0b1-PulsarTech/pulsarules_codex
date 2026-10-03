package packagefilecount

import (
	"go/ast"
	"path"
	"strconv"
)

const entImportPath = "entgo.io/ent"

func inspectEnt(file *ast.File) bool {
	locals := make(map[string]bool)
	for _, spec := range file.Imports {
		got, err := strconv.Unquote(spec.Path.Value)
		if err != nil || got != entImportPath && got != entImportPath+"/schema/mixin" {
			continue
		}
		local := path.Base(got)
		if spec.Name != nil {
			local = spec.Name.Name
		}
		locals[local] = true
	}
	return embedsSchema(file, locals)
}

func embedsSchema(file *ast.File, locals map[string]bool) bool {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, isType := spec.(*ast.TypeSpec)
			if !isType {
				continue
			}
			if st, isStruct := ts.Type.(*ast.StructType); isStruct && hasSchemaField(st, locals) {
				return true
			}
		}
	}
	return false
}

func hasSchemaField(st *ast.StructType, locals map[string]bool) bool {
	for _, field := range st.Fields.List {
		sel, isSel := field.Type.(*ast.SelectorExpr)
		if len(field.Names) != 0 || !isSel || sel.Sel.Name != "Schema" {
			continue
		}
		if pkg, isIdent := sel.X.(*ast.Ident); isIdent && locals[pkg.Name] {
			return true
		}
	}
	return false
}
