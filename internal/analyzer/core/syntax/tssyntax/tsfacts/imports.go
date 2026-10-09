package tsfacts

import (
	"slices"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsalias"
)

const (
	groupBuiltin = iota
	groupExternal
	groupLocal
)

func (coll *collector) imports(statements []*ast.Node) {
	for _, stmt := range statements {
		spec := importSpecifier(stmt)
		if spec == nil {
			continue
		}
		coll.out.Imports = append(coll.out.Imports, syntax.Import{
			Literal: coll.sourceOf(spec),
			Line:    coll.lineAt(stmt.Pos()),
			Group:   coll.importGroup(spec),
		})
	}
	if len(coll.out.Imports) > 0 {
		coll.out.ImportGroups = importGroupNames()
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

func (coll *collector) importGroup(spec *ast.Node) int {
	if spec.Kind != ast.KindStringLiteral {
		return groupExternal
	}
	path := spec.Text()
	switch {
	case strings.HasPrefix(path, "node:"), strings.HasPrefix(path, "bun:"):
		return groupBuiltin
	case strings.HasPrefix(path, "."), strings.HasPrefix(path, "/"):
		return groupLocal
	case isAliasSpelling(path), coll.matchesLocal(path):
		return groupLocal
	}
	return groupExternal
}

func isAliasSpelling(path string) bool {
	return strings.HasPrefix(path, "#") || strings.HasPrefix(path, "@/") ||
		strings.HasPrefix(path, "~/") || path == "$lib" || strings.HasPrefix(path, "$lib/")
}

func (coll *collector) matchesLocal(path string) bool {
	return slices.ContainsFunc(coll.local, func(pattern string) bool {
		_, ok := tsalias.Match(pattern, path)
		return ok
	})
}
