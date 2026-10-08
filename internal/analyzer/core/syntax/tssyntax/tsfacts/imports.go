package tsfacts

import (
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

const (
	groupBuiltin = iota
	groupPackage
	groupRelative
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
			Group:   importGroup(spec),
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
	names := make([]string, groupRelative+1)
	names[groupBuiltin] = "node:"
	names[groupPackage] = "package"
	names[groupRelative] = "relative"
	return names
}

func importGroup(spec *ast.Node) int {
	if spec.Kind != ast.KindStringLiteral {
		return groupPackage
	}
	path := spec.Text()
	switch {
	case strings.HasPrefix(path, "node:"):
		return groupBuiltin
	case strings.HasPrefix(path, "."), strings.HasPrefix(path, "/"):
		return groupRelative
	}
	return groupPackage
}
