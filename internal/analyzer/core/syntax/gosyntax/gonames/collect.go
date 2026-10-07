package gonames

import (
	"go/ast"
	"go/token"
	"unicode"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type collector struct {
	fset     *token.FileSet
	seen     map[token.Pos]bool
	names    []syntax.Name
	testPkgs map[string]bool
}

func Collect(fset *token.FileSet, file *ast.File) []syntax.Name {
	scan := &collector{
		fset:     fset,
		seen:     map[token.Pos]bool{},
		testPkgs: testingNames(file),
	}
	scan.add(file.Name, syntax.NamePackage)
	ast.Inspect(file, scan.visit)
	return scan.names
}

func (scan *collector) add(ident *ast.Ident, kind syntax.NameKind) *syntax.Name {
	if ident == nil || scan.seen[ident.Pos()] {
		return nil
	}
	scan.seen[ident.Pos()] = true
	scan.names = append(scan.names, syntax.Name{
		Name:     ident.Name,
		Kind:     kind,
		Line:     scan.fset.Position(ident.Pos()).Line,
		Exported: isExported(ident.Name, kind),
	})
	return &scan.names[len(scan.names)-1]
}

func isExported(name string, kind syntax.NameKind) bool {
	letter, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(letter) && canBeExported(kind)
}

func canBeExported(kind syntax.NameKind) bool {
	switch kind {
	case syntax.NameFunction, syntax.NameMethod, syntax.NameType, syntax.NameConstant,
		syntax.NameVariable, syntax.NameField:
		return true
	}
	return false
}

func (scan *collector) addFields(list *ast.FieldList, kind syntax.NameKind) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		exempt := kind == syntax.NameParameter && isTestingType(field.Type, scan.testPkgs)
		for _, ident := range field.Names {
			if name := scan.add(ident, kind); name != nil {
				name.Exempt = exempt
			}
		}
	}
}

func (scan *collector) visit(node ast.Node) bool {
	scan.declSite(node)
	scan.stmtSite(node)
	return true
}
