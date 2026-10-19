package shadowing

import "go/types"

const (
	originVariable  = "variable"
	originSignature = "signature"
)

type scopeEntry struct {
	name   string
	origin string
}

type scopeStack struct {
	stack [][]scopeEntry
}

func newScopeStack() *scopeStack {
	return &scopeStack{}
}

func (scopes *scopeStack) push() {
	scopes.stack = append(scopes.stack, nil)
}

func (scopes *scopeStack) pop() {
	if len(scopes.stack) > 0 {
		scopes.stack = scopes.stack[:len(scopes.stack)-1]
	}
}

func (scopes *scopeStack) declare(name string) {
	scopes.declareAs(name, originVariable)
}

func (scopes *scopeStack) declareAs(name, origin string) {
	if len(scopes.stack) == 0 {
		return
	}
	cur := &scopes.stack[len(scopes.stack)-1]
	for idx := range *cur {
		if (*cur)[idx].name == name {
			// Already declared in this scope.
			return
		}
	}
	*cur = append(*cur, scopeEntry{name: name, origin: origin})
}

func (scopes *scopeStack) lookup(name string) (string, bool) {
	const originBuiltin = "builtin"
	if types.Universe.Lookup(name) != nil {
		return originBuiltin, true
	}
	// Every scope but the innermost is checked.
	for idx := 0; idx < len(scopes.stack)-1; idx++ {
		for _, entry := range scopes.stack[idx] {
			if entry.name == name {
				return entry.origin, true
			}
		}
	}
	return "", false
}

func (scopes *scopeStack) declaredHere(name string) (string, bool) {
	if len(scopes.stack) == 0 {
		return "", false
	}
	// Go rejects a second declaration of a name in one block, so a hit means `:=` reassigns.
	for _, entry := range scopes.stack[len(scopes.stack)-1] {
		if entry.name == name {
			return entry.origin, true
		}
	}
	return "", false
}
