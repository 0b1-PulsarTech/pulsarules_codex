package gosyntax

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func imports(fset *token.FileSet, file *ast.File, modulePath string) ([]syntax.Import, []string) {
	if len(file.Imports) == 0 || modulePath == "" {
		return nil, nil
	}
	found := make([]syntax.Import, 0, len(file.Imports))
	for _, spec := range file.Imports {
		found = append(found, syntax.Import{
			Literal: spec.Path.Value,
			Line:    lineOf(fset, spec.Pos()),
			Group:   classify(spec.Path.Value, modulePath),
		})
	}
	return found, []string{"stdlib", "external", modulePath}
}

func classify(literal, modulePath string) int {
	const (
		groupStdlib = iota
		groupExternal
		groupModule
	)
	clean := strings.Trim(literal, `"`)
	if clean == modulePath || strings.HasPrefix(clean, modulePath+"/") {
		return groupModule
	}
	if strings.Contains(clean, ".") {
		return groupExternal
	}
	return groupStdlib
}
