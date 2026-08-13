package naming

import (
	"go/ast"
	"go/token"
	"strings"
)

// fileIndex is what a single identifier cannot answer on its own: whether it
// has numbered siblings, and whether it merely echoes the type it was built
// from. Both questions need the whole file, so it is collected once up front.
type fileIndex struct {
	// declared holds every identifier the file declares, used to decide
	// whether a numbered name has a sibling worth calling a sequence.
	declared map[string]bool
	// typeDerived holds the BINDING SITES whose name repeats the type they
	// come from - a constructor result, composite literal, or parameter
	// type; such a name is precise, not noise. Keyed by position, not name:
	// a file-wide exemption would let one `manager := NewManager()` clear
	// every other `manager` in the file.
	typeDerived map[token.Pos]bool
}

func newFileIndex(f *ast.File) *fileIndex {
	idx := &fileIndex{
		declared:    make(map[string]bool),
		typeDerived: make(map[token.Pos]bool),
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			idx.addFuncDecl(v)
		case *ast.GenDecl:
			idx.addGenDecl(v)
		case *ast.AssignStmt:
			idx.addAssign(v)
		}
		return true
	})
	return idx
}

func (idx *fileIndex) addFuncDecl(fn *ast.FuncDecl) {
	if fn.Name != nil {
		idx.declared[fn.Name.Name] = true
	}
	for _, list := range []*ast.FieldList{fn.Recv, fn.Type.Params, fn.Type.Results} {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, id := range field.Names {
				idx.declared[id.Name] = true
				idx.markIfTypeDerived(id, typeName(field.Type))
			}
		}
	}
}

func (idx *fileIndex) addGenDecl(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			for pos, id := range s.Names {
				idx.declared[id.Name] = true
				idx.markIfTypeDerived(id, typeName(s.Type))
				// Values line up with Names only when the spec has one of
				// each; anything else is a single call feeding the group.
				if len(s.Values) == len(s.Names) {
					idx.markIfTypeDerived(id, typeOfExpr(s.Values[pos]))
				}
			}
		case *ast.TypeSpec:
			// A type declaration is NOT exempted by its own name: `type Data
			// struct{}` is exactly the vague name the rule exists to catch.
			if s.Name != nil {
				idx.declared[s.Name.Name] = true
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
		idx.typeDerived[id.NamePos] = true
	}
}

// typeOfExpr names the type an expression yields, as far as syntax alone can
// tell: a composite literal wears its type, and a constructor call carries it
// in the function name. No go/types here - loading packages would cost more
// than this rule is worth.
func typeOfExpr(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.CompositeLit:
		return typeName(v.Type)
	case *ast.CallExpr:
		return strings.TrimPrefix(typeName(v.Fun), "New")
	case *ast.UnaryExpr:
		return typeOfExpr(v.X)
	}
	return ""
}

// typeName reduces a type expression to its bare name, dropping the pointer,
// the package qualifier, the slice brackets and the generic arguments.
func typeName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return typeName(v.X)
	case *ast.SelectorExpr:
		if v.Sel == nil {
			return ""
		}
		return v.Sel.Name
	case *ast.IndexExpr:
		return typeName(v.X)
	case *ast.IndexListExpr:
		return typeName(v.X)
	case *ast.ArrayType:
		return typeName(v.Elt)
	}
	return ""
}

// hasNumberedSibling reports whether stem also appears in the file bare, or
// with a DIFFERENT low counter. That pair is the actual smell the rule is
// after - copy-paste siblings - which one digit on its own never proves.
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

// why: a counter series can start at zero (user0/user1), so the floor is 0.
func isCounterValue(value int) bool {
	return value >= 0 && value <= maxCounterValue
}
