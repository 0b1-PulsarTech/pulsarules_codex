package gosrc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

func topLevelNames(file *ast.File, set map[string]bool) {
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if typed.Recv == nil {
				set[typed.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				addSpecNames(spec, set)
			}
		}
	}
}

func addSpecNames(spec ast.Spec, set map[string]bool) {
	switch typed := spec.(type) {
	case *ast.TypeSpec:
		set[typed.Name.Name] = true
	case *ast.ValueSpec:
		for _, name := range typed.Names {
			set[name.Name] = true
		}
	}
}

func PackageClause(src string) (name string, isIgnored bool) {
	file, err := parser.ParseFile(
		token.NewFileSet(), "", src, parser.PackageClauseOnly|parser.ParseComments,
	)
	if err != nil {
		return "", false
	}
	return file.Name.Name, gomod.IsBuildIgnored(file)
}

func exampleBodies(file *ast.File, tokFile *token.File) [][2]int {
	var spans [][2]int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Body == nil || !isExampleName(fn.Name.Name) {
			continue
		}
		spans = append(spans, [2]int{tokFile.Offset(fn.Body.Pos()), tokFile.Offset(fn.Body.End())})
	}
	return spans
}

func isExampleName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Example")
	if !ok || rest == "" {
		return ok
	}
	first, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(first)
}

func (goFile File) inExample(off int) bool {
	return slices.ContainsFunc(goFile.examples, func(span [2]int) bool {
		return off >= span[0] && off < span[1]
	})
}
