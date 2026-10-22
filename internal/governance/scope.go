package governance

import (
	"fmt"
	"strings"
)

type Scope int

const (
	ScopeFull Scope = iota
	ScopeCommit
	ScopeChanged
)

type unknownScopeError struct {
	Name string
}

func (err unknownScopeError) Error() string {
	return fmt.Sprintf("%q (want %s)", err.Name, strings.Join(scopeNames(), "|"))
}

type namedScope struct {
	name  string
	scope Scope
}

func scopeTable() []namedScope {
	return []namedScope{
		{"full", ScopeFull},
		{"commit", ScopeCommit},
		{"changed", ScopeChanged},
	}
}

func scopeNames() []string {
	table := scopeTable()
	names := make([]string, len(table))
	for idx, listed := range table {
		names[idx] = listed.name
	}
	return names
}

func ParseScope(raw string) (Scope, error) {
	if raw == "" {
		return ScopeFull, nil
	}
	for _, listed := range scopeTable() {
		if listed.name == raw {
			return listed.scope, nil
		}
	}
	return ScopeFull, unknownScopeError{Name: raw}
}
