package naming

import (
	"fmt"
	"go/ast"
	"go/token"
)

// fileChecker runs the naming rules over one file. It holds the file index and
// the emit callback as fields so each rule reads one identifier plus the
// context it genuinely needs, instead of growing the call signature.
type fileChecker struct {
	index *fileIndex
	emit  func(pos token.Pos, msg, suggestion string)
}

func (c fileChecker) funcDecl(v *ast.FuncDecl) {
	if v.Name != nil {
		c.check(v.Name.Name, v.Name.NamePos)
	}
	if v.Type.Params != nil {
		c.checkFields(v.Type.Params)
	}
	if v.Type.Results != nil {
		c.checkFields(v.Type.Results)
	}
}

func (c fileChecker) checkFields(list *ast.FieldList) {
	for _, field := range list.List {
		for _, id := range field.Names {
			c.check(id.Name, id.NamePos)
		}
	}
}

func (c fileChecker) genDecl(v *ast.GenDecl) {
	for _, spec := range v.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for _, id := range s.Names {
				c.check(id.Name, id.NamePos)
			}
		case *ast.TypeSpec:
			if s.Name != nil {
				c.check(s.Name.Name, s.Name.NamePos)
			}
		}
	}
}

// check runs all naming rules against one identifier.
func (c fileChecker) check(name string, pos token.Pos) {
	if name == "_" || name == "" {
		return
	}

	if isExported(name) && len(name) == 1 {
		c.emit(pos, fmt.Sprintf("single-letter exported name %q", name), "use a descriptive name")
	}

	if c.isSequential(name) {
		c.emit(
			pos,
			fmt.Sprintf("numbered name %q suggests sequential naming", name),
			"use a descriptive name instead of a number suffix",
		)
	}

	if checkHungarian(name) {
		c.emit(
			pos,
			fmt.Sprintf("name %q appears to use Hungarian notation", name),
			"use plain descriptive names without type prefixes",
		)
	}

	if c.isNoise(name, pos) {
		c.emit(
			pos,
			fmt.Sprintf("name %q contains a noise word", name),
			"use a more specific descriptive name",
		)
	}
}

// isSequential reports the copy-paste-siblings smell: a LOW counter suffix
// that some other name in the file actually counts against. A lone digit
// proves nothing - sha256, limit32 and oauth2 all carry a number that IS the
// concept - so the sibling is what turns a digit into a sequence.
func (c fileChecker) isSequential(name string) bool {
	stem, value, ok := numberedStem(name)
	if !ok || !isCounterValue(value) {
		return false
	}
	return c.index.hasNumberedSibling(name, stem)
}

// isNoise reports a bare noise word, unless THIS binding repeats the type it
// was built from: `manager := NewManager()` names its value after exactly what
// it is, which is the opposite of vague. The exemption is per binding site, so
// a `manager := find()` elsewhere in the file still fires.
func (c fileChecker) isNoise(name string, pos token.Pos) bool {
	return checkNoiseWord(name) && !c.index.typeDerived[pos]
}
