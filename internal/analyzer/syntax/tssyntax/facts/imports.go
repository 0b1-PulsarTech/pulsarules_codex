package facts

import (
	"slices"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/pathalias"
)

const (
	groupBuiltin = iota
	groupExternal
	groupLocal
)

type importFacts struct {
	source
	local []string
	out   []syntax.Import
}

func (imports *importFacts) collect(statements []*ast.Node) {
	for _, stmt := range statements {
		spec := importSpecifier(stmt)
		if spec == nil {
			continue
		}
		imports.out = append(imports.out, syntax.Import{
			Literal: imports.sourceOf(spec),
			Line:    imports.lineAt(stmt.Pos()),
			Group:   imports.importGroup(spec),
		})
	}
}

func importSpecifier(stmt *ast.Node) *ast.Node {
	switch stmt.Kind {
	case ast.KindImportDeclaration:
		return stmt.AsImportDeclaration().ModuleSpecifier
	case ast.KindImportEqualsDeclaration:
		reference := stmt.AsImportEqualsDeclaration().ModuleReference
		if reference != nil && reference.Kind == ast.KindExternalModuleReference {
			return reference.Expression()
		}
	}
	return nil
}

func importGroupNames() []string {
	names := make([]string, groupLocal+1)
	names[groupBuiltin] = "node:"
	names[groupExternal] = "external"
	names[groupLocal] = "local"
	return names
}

func (imports *importFacts) importGroup(spec *ast.Node) int {
	if spec.Kind != ast.KindStringLiteral {
		return groupExternal
	}
	path := spec.Text()
	switch {
	case strings.HasPrefix(path, "node:"), strings.HasPrefix(path, "bun:"):
		return groupBuiltin
	case strings.HasPrefix(path, "."), strings.HasPrefix(path, "/"):
		return groupLocal
	case isAliasSpelling(path), imports.matchesLocal(path):
		return groupLocal
	}
	return groupExternal
}

func isAliasSpelling(path string) bool {
	return strings.HasPrefix(path, "#") || strings.HasPrefix(path, "@/") ||
		strings.HasPrefix(path, "~/") || path == "$lib" || strings.HasPrefix(path, "$lib/")
}

func (imports *importFacts) matchesLocal(path string) bool {
	return slices.ContainsFunc(imports.local, func(pattern string) bool {
		_, ok := pathalias.Match(pattern, path)
		return ok
	})
}
