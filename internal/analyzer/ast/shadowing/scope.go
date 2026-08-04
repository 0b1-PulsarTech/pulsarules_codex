package shadowing

// origin values recorded on a scope entry. originSignature is what separates a
// reassignment from a shadow: a name bound by the function signature lives in
// the SAME block as the body's top-level statements, so a `:=` naming it
// reassigns rather than declares.
const (
	originVariable  = "variable"
	originBuiltin   = "builtin"
	originSignature = "signature"
)

// scopeEntry records one declared name and its origin description.
type scopeEntry struct {
	name   string
	origin string // combined with "outer" by the caller
}

// scopeStack maintains nested variable scopes for one function.
type scopeStack struct {
	stack [][]scopeEntry
}

func newScopeStack() *scopeStack {
	s := &scopeStack{}
	// outermost builtins scope
	s.push()
	s.stack[0] = builtins()
	return s
}

func (s *scopeStack) push() {
	s.stack = append(s.stack, nil)
}

func (s *scopeStack) pop() {
	if len(s.stack) > 1 {
		s.stack = s.stack[:len(s.stack)-1]
	}
}

func (s *scopeStack) declare(name string) {
	s.declareAs(name, originVariable)
}

func (s *scopeStack) declareAs(name, origin string) {
	if len(s.stack) == 0 {
		return
	}
	cur := &s.stack[len(s.stack)-1]
	for i := range *cur {
		if (*cur)[i].name == name {
			return // already declared in this scope
		}
	}
	*cur = append(*cur, scopeEntry{name: name, origin: origin})
}

// lookup searches all scopes from outermost to innermost (excluding the
// current scope) for a matching name. It returns the origin string if found.
func (s *scopeStack) lookup(name string) (string, bool) {
	// check all scopes up to but NOT including the innermost (current) one
	// also: builtins in stack[0] are included
	for i := 0; i < len(s.stack)-1; i++ {
		for _, e := range s.stack[i] {
			if e.name == name {
				return e.origin, true
			}
		}
	}
	return "", false
}

// declaredHere reports the origin of a name already bound in the CURRENT
// scope. Go rejects a second declaration of the same name in one block, so a
// hit here means the `:=` reassigns an existing binding instead of creating a
// new one.
func (s *scopeStack) declaredHere(name string) (string, bool) {
	if len(s.stack) == 0 {
		return "", false
	}
	for _, e := range s.stack[len(s.stack)-1] {
		if e.name == name {
			return e.origin, true
		}
	}
	return "", false
}

func builtins() []scopeEntry {
	names := []string{
		"true", "false", "nil",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "float32", "float64",
		"complex64", "complex128",
		"string", "bool", "byte", "rune",
		"error",
		"any", "comparable",
		"make", "new", "len", "cap", "append",
		"copy", "delete", "close",
		"panic", "recover",
		"print", "println",
		"real", "imag", "complex",
		"iota",
	}
	entries := make([]scopeEntry, len(names))
	for i, n := range names {
		entries[i] = scopeEntry{name: n, origin: originBuiltin}
	}
	return entries
}
