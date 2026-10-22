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
	return ScopeFull, fmt.Errorf("%q (want %s)", raw, strings.Join(scopeNames(), "|"))
}
