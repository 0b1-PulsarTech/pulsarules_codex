package analysis

type Scope int

const (
	ScopeFull Scope = iota
	ScopeCommit
	ScopeChanged
)

func ParseScope(raw string) Scope {
	switch raw {
	case "commit":
		return ScopeCommit
	default:
		return ScopeFull
	}
}
