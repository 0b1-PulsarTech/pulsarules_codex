package identifier

import (
	"go/ast"
	"go/token"
	"strings"
)

type binding struct {
	name string
	pos  token.Pos
}

type fileIndex struct {
	declared    map[string]bool
	typeDerived map[token.Pos]bool
	// checked lists, in source order, the names the rules judge; declared also holds
	// receivers and plain assignments, which only count as siblings.
	checked []binding
}

func newFileIndex(file *ast.File) *fileIndex {
	idx := &fileIndex{
		declared:    make(map[string]bool),
		typeDerived: make(map[token.Pos]bool),
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			idx.addFuncDecl(typed)
		case *ast.GenDecl:
			idx.addGenDecl(typed)
		case *ast.AssignStmt:
			idx.addAssign(typed)
		}
		return true
	})
	return idx
}

func (idx *fileIndex) addChecked(id *ast.Ident) {
	idx.checked = append(idx.checked, binding{name: id.Name, pos: id.NamePos})
}

func (idx *fileIndex) addFuncDecl(fn *ast.FuncDecl) {
	if fn.Name != nil {
		idx.declared[fn.Name.Name] = true
		idx.addChecked(fn.Name)
	}
	lists := []*ast.FieldList{fn.Recv, fn.Type.Params, fn.Type.Results}
	for slot, list := range lists {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, id := range field.Names {
				idx.declared[id.Name] = true
				idx.markIfTypeDerived(id, typeName(field.Type))
				// A receiver is indexed as a sibling but never judged.
				if slot != 0 {
					idx.addChecked(id)
				}
			}
		}
	}
}

func (idx *fileIndex) addGenDecl(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.ValueSpec:
			for pos, id := range typed.Names {
				idx.declared[id.Name] = true
				idx.addChecked(id)
				idx.markIfTypeDerived(id, typeName(typed.Type))
				// Values line up with Names only when the spec has one of
				// each; anything else is a single call feeding the group.
				if len(typed.Values) == len(typed.Names) {
					idx.markIfTypeDerived(id, typeOfExpr(typed.Values[pos]))
				}
			}
		case *ast.TypeSpec:
			// A type declaration is not exempted by its own name: `type Data
			// struct{}` is exactly the vague name the rule exists to catch.
			if typed.Name != nil {
				idx.declared[typed.Name.Name] = true
				idx.addChecked(typed.Name)
			}
		}
	}
}

func (idx *fileIndex) addAssign(assign *ast.AssignStmt) {
	for pos, lhs := range assign.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		idx.declared[id.Name] = true
		if assign.Tok == token.DEFINE {
			idx.addChecked(id)
		}
		switch {
		case len(assign.Rhs) == len(assign.Lhs):
			idx.markIfTypeDerived(id, typeOfExpr(assign.Rhs[pos]))
		case len(assign.Rhs) == 1:
			// One call feeding several names: every name sees the same type.
			idx.markIfTypeDerived(id, typeOfExpr(assign.Rhs[0]))
		}
	}
}

func (idx *fileIndex) markIfTypeDerived(id *ast.Ident, derivedFrom string) {
	if derivedFrom != "" && strings.EqualFold(id.Name, derivedFrom) {
		// Keyed by position, not name: a file-wide exemption would let one
		// `manager := NewManager()` clear every other `manager`.
		idx.typeDerived[id.NamePos] = true
	}
}

func typeOfExpr(expr ast.Expr) string {
	// No go/types: loading packages costs more than this rule is worth.
	switch typed := expr.(type) {
	case *ast.CompositeLit:
		return typeName(typed.Type)
	case *ast.CallExpr:
		return strings.TrimPrefix(typeName(typed.Fun), "New")
	case *ast.UnaryExpr:
		return typeOfExpr(typed.X)
	}
	return ""
}

func typeName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return typeName(typed.X)
	case *ast.SelectorExpr:
		if typed.Sel == nil {
			return ""
		}
		return typed.Sel.Name
	case *ast.IndexExpr:
		return typeName(typed.X)
	case *ast.IndexListExpr:
		return typeName(typed.X)
	case *ast.ArrayType:
		return typeName(typed.Elt)
	}
	return ""
}

func (idx *fileIndex) hasNumberedSibling(name, stem string) bool {
	if idx.declared[stem] {
		return true
	}
	for other := range idx.declared {
		if other == name {
			continue
		}
		if otherStem, otherValue, ok := numberedStem(other); ok &&
			otherStem == stem && isCounterValue(otherValue) {
			return true
		}
	}
	return false
}

func isCounterValue(value int) bool {
	const maxCounterValue = 7
	// simplification: sequences over 7 report only the first 7. Upgrade path:
	// compare digit run length, require siblings to form a contiguous series.
	// Machine vocabulary (bit widths, hashes) starts at 8; a counter series can start at 0.
	return value >= 0 && value <= maxCounterValue
}
