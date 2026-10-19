package filecount

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
)

func inspectEnt(file *ast.File) bool {
	const entImportPath = "entgo.io/ent"
	locals := make(map[string]bool)
	for _, importPath := range []string{entImportPath, entImportPath + "/schema/mixin"} {
		for _, local := range goast.ImportNames(file, importPath) {
			locals[local] = true
		}
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
