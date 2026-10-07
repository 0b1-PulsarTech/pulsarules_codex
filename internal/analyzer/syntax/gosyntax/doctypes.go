package gosyntax

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func typeTargets(gen *ast.GenDecl) []docTarget {
	symbol := "type group"
	if !gen.Lparen.IsValid() && len(gen.Specs) > 0 {
		symbol = specName(gen.Specs[0])
	}
	targets := []docTarget{{symbol: symbol, kind: syntax.DocType, group: gen.Doc}}
	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		targets = append(targets, docTarget{
			symbol: typeSpec.Name.Name, kind: syntax.DocType, group: typeSpec.Doc,
		})
		targets = append(targets, methodTargets(typeSpec)...)
	}
	return targets
}

func specName(spec ast.Spec) string {
	if typeSpec, ok := spec.(*ast.TypeSpec); ok {
		return typeSpec.Name.Name
	}
	return "type declaration"
}

func methodTargets(typeSpec *ast.TypeSpec) []docTarget {
	iface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok || iface.Methods == nil {
		return nil
	}
	var targets []docTarget
	for _, field := range iface.Methods.List {
		name := "embedded interface"
		if len(field.Names) > 0 {
			name = field.Names[0].Name
		}
		targets = append(targets, docTarget{
			symbol: typeSpec.Name.Name + "." + name,
			kind:   syntax.DocMethod,
			group:  field.Doc,
		})
	}
	return targets
}
