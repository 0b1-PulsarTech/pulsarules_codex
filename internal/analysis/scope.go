package analysis

// Scope controls which analyzers run and what context is built.
type Scope int

const (
	// ScopeFull runs all registered analyzers with full project context.
	ScopeFull Scope = iota
	// ScopeCommit runs static + AST + commit analyzers for pre-commit hooks.
	ScopeCommit
	// ScopeChanged runs static + AST + arch analyzers over the changed
	// files, skipping external-tool delegation (golangci-lint).
	ScopeChanged
)

// ParseScope converts a scope flag string to a Scope value. Unrecognized
// values, including the empty string, default to ScopeFull.
func ParseScope(s string) Scope {
	switch s {
	case "commit":
		return ScopeCommit
	case "changed":
		return ScopeChanged
	default:
		return ScopeFull
	}
}
