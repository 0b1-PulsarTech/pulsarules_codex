package shadowing

import "go/token"

// scopeEntry records one declared name and its origin description.
type scopeEntry struct {
	name   string
	origin string // "variable", "builtin" - combined with "outer" by the caller
}

// scopeStack maintains nested variable scopes for one function.
type scopeStack struct {
	stack [][]scopeEntry
}

func newScopeStack() *scopeStack {
	s := &scopeStack{}
	// innermost builtins scope
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

func (s *scopeStack) declare(name string, _ token.Pos) {
	if len(s.stack) == 0 {
		return
	}
	cur := &s.stack[len(s.stack)-1]
	for i := range *cur {
		if (*cur)[i].name == name {
			return // already declared in this scope
		}
	}
	*cur = append(*cur, scopeEntry{name: name, origin: "variable"})
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
		entries[i] = scopeEntry{name: n, origin: "builtin"}
	}
	return entries
}
