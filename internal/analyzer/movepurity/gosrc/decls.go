package gosrc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"unicode"
)

func CollectDecls(src, pkg string, set map[string]bool) {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != pkg {
		return
	}
	topLevelNames(file, set)
}

func CollectAnyDecls(src string, set map[string]bool) {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution)
	if err != nil {
		return
	}
	topLevelNames(file, set)
}

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
	return file.Name.Name, hasIgnoreConstraint(file)
}

func hasIgnoreConstraint(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.End() > file.Package {
			return false
		}
		for _, comment := range group.List {
			isConstraint := strings.HasPrefix(comment.Text, "//go:build") ||
				strings.HasPrefix(comment.Text, "// +build")
			words := strings.FieldsFunc(comment.Text, func(char rune) bool {
				return !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_'
			})
			if isConstraint && slices.Contains(words, "ignore") {
				return true
			}
		}
	}
	return false
}
