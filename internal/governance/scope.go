package governance

type Scope int

const (
	ScopeFull Scope = iota
	ScopeCommit
	ScopeChanged
)
