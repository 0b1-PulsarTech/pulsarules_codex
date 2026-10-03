package identifier

import (
	"fmt"
	"go/ast"
	"go/token"
)

type fileChecker struct {
	index *fileIndex
	emit  func(pos token.Pos, msg, suggestion string)
}

func (checker fileChecker) funcDecl(fn *ast.FuncDecl) {
	if fn.Name != nil {
		checker.check(fn.Name.Name, fn.Name.NamePos)
	}
	if fn.Type.Params != nil {
		checker.checkFields(fn.Type.Params)
	}
	if fn.Type.Results != nil {
		checker.checkFields(fn.Type.Results)
	}
}

func (checker fileChecker) checkFields(list *ast.FieldList) {
	for _, field := range list.List {
		for _, id := range field.Names {
			checker.check(id.Name, id.NamePos)
		}
	}
}

func (checker fileChecker) genDecl(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range typed.Names {
				checker.check(id.Name, id.NamePos)
			}
		case *ast.TypeSpec:
			if typed.Name != nil {
				checker.check(typed.Name.Name, typed.Name.NamePos)
			}
		}
	}
}

func (checker fileChecker) check(name string, pos token.Pos) {
	if name == "_" || name == "" {
		return
	}

	if isExported(name) && len(name) == 1 {
		checker.emit(
			pos, fmt.Sprintf("single-letter exported name %q", name), "use a descriptive name",
		)
	}

	if checker.isSequential(name) {
		checker.emit(
			pos,
			fmt.Sprintf("numbered name %q suggests sequential naming", name),
			"use a descriptive name instead of a number suffix",
		)
	}

	if checkHungarian(name) {
		checker.emit(
			pos,
			fmt.Sprintf("name %q appears to use Hungarian notation", name),
			"use plain descriptive names without type prefixes",
		)
	}

	if checker.isNoise(name, pos) {
		checker.emit(
			pos,
			fmt.Sprintf("name %q contains a noise word", name),
			"use a more specific descriptive name",
		)
	}
}

func (checker fileChecker) isSequential(name string) bool {
	stem, value, ok := numberedStem(name)
	if !ok || !isCounterValue(value) {
		return false
	}
	// A lone digit proves nothing (sha256, oauth2); a sibling makes it a sequence.
	return checker.index.hasNumberedSibling(name, stem)
}

func (checker fileChecker) isNoise(name string, pos token.Pos) bool {
	// A binding that repeats the type it was built from is exempt, per binding site.
	return checkNoiseWord(name) && !checker.index.typeDerived[pos]
}
