package goast

import (
	"go/ast"
	"path"
	"strconv"
)

const (
	blankImport = "_"
	dotImport   = "."
)

func ImportNames(file *ast.File, importPath string) []string {
	var names []string
	for _, spec := range file.Imports {
		got, err := strconv.Unquote(spec.Path.Value)
		if err != nil || got != importPath {
			continue
		}
		switch {
		case spec.Name == nil:
			names = append(names, path.Base(got))
		case spec.Name.Name != blankImport:
			names = append(names, spec.Name.Name)
		}
	}
	return names
}

func AllImportNames(file *ast.File) map[string]bool {
	names := make(map[string]bool, len(file.Imports))
	for _, spec := range file.Imports {
		got, err := strconv.Unquote(spec.Path.Value)
		switch {
		case err != nil:
		case spec.Name == nil:
			names[path.Base(got)] = true
		case spec.Name.Name != blankImport && spec.Name.Name != dotImport:
			names[spec.Name.Name] = true
		}
	}
	return names
}
