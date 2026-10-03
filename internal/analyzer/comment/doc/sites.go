package doc

import (
	"go/ast"
	"strings"
)

type docSite struct {
	symbol string
	group  *ast.CommentGroup
	hint   string
}

func typeSites(gen *ast.GenDecl) []docSite {
	const typeHint = "delete it; the type name and fields carry the meaning"
	symbol := "type group"
	if !gen.Lparen.IsValid() && len(gen.Specs) > 0 {
		symbol = specName(gen.Specs[0])
	}
	sites := []docSite{{symbol: symbol, group: gen.Doc, hint: typeHint}}
	for _, spec := range gen.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		sites = append(
			sites,
			docSite{symbol: typeSpec.Name.Name, group: typeSpec.Doc, hint: typeHint},
		)
		sites = append(sites, methodSites(typeSpec)...)
	}
	return sites
}

func specName(spec ast.Spec) string {
	if typeSpec, ok := spec.(*ast.TypeSpec); ok {
		return typeSpec.Name.Name
	}
	return "type declaration"
}

func methodSites(typeSpec *ast.TypeSpec) []docSite {
	const methodHint = "delete it; the method name and signature carry the meaning"
	iface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok || iface.Methods == nil {
		return nil
	}
	var sites []docSite
	for _, field := range iface.Methods.List {
		name := "embedded interface"
		if len(field.Names) > 0 {
			name = field.Names[0].Name
		}
		sites = append(sites, docSite{
			symbol: typeSpec.Name.Name + "." + name,
			group:  field.Doc,
			hint:   methodHint,
		})
	}
	return sites
}

func isDeprecatedOnly(group *ast.CommentGroup) bool {
	text := strings.TrimSpace(group.Text())
	const maxLines = 3
	return strings.HasPrefix(text, "Deprecated:") && !strings.Contains(text, "\n\n") &&
		strings.Count(text, "\n")+1 <= maxLines
}
