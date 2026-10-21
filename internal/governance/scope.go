package governance

import "fmt"

type Scope int

const (
	ScopeFull Scope = iota
	ScopeCommit
	ScopeChanged
)

func ParseScope(raw string) (Scope, error) {
	switch raw {
	case "", "full":
		return ScopeFull, nil
	case "commit":
		return ScopeCommit, nil
	case "changed":
		return ScopeChanged, nil
	}
	return ScopeFull, fmt.Errorf("invalid --scope %q (want full|commit|changed)", raw)
}
