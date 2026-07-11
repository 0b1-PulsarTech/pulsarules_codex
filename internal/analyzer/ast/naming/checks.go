package naming

import (
	"fmt"
	"go/ast"
	"go/token"
)

func checkFuncDecl(v *ast.FuncDecl, emit func(token.Pos, string, string)) {
	if v.Name != nil {
		checkName(v.Name.Name, v.Name.NamePos, emit)
	}
	for _, p := range v.Type.Params.List {
		for _, id := range p.Names {
			checkName(id.Name, id.NamePos, emit)
		}
	}
	if v.Type.Results != nil {
		for _, r := range v.Type.Results.List {
			for _, id := range r.Names {
				checkName(id.Name, id.NamePos, emit)
			}
		}
	}
}

func checkGenDecl(v *ast.GenDecl, emit func(token.Pos, string, string)) {
	for _, spec := range v.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range s.Names {
				checkName(id.Name, id.NamePos, emit)
			}
		case *ast.TypeSpec:
			if s.Name != nil {
				checkName(s.Name.Name, s.Name.NamePos, emit)
			}
		}
	}
}

// checkName runs all naming rules against one identifier.
func checkName(name string, pos token.Pos, emit func(token.Pos, string, string)) {
	if name == "_" || name == "" {
		return
	}

	if isExported(name) && len(name) == 1 {
		emit(pos, fmt.Sprintf("single-letter exported name %q", name), "use a descriptive name")
	}

	if checkNumbered(name) {
		emit(
			pos,
			fmt.Sprintf("numbered name %q suggests sequential naming", name),
			"use a descriptive name instead of a number suffix",
		)
	}

	if checkHungarian(name) {
		emit(
			pos,
			fmt.Sprintf("name %q appears to use Hungarian notation", name),
			"use plain descriptive names without type prefixes",
		)
	}

	if checkNoiseWord(name) {
		emit(
			pos,
			fmt.Sprintf("name %q contains a noise word", name),
			"use a more specific descriptive name",
		)
	}

	checkDisinformation(name)
}
