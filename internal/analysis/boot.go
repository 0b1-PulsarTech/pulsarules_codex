package analysis

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/golang"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// analyzerBuilder constructs an analyzer at registration time, closing over
// the rule index, the shared language registry, and the repository the
// move-purity analyzer reads staged diffs through (nil when no repository
// is available).
type analyzerBuilder func(
	index *knowledge.Index,
	langs *core.LanguageRegistry,
	repo vcs.Repository,
) core.Analyzer

// analyzerSpec pairs an analyzer builder with the scopes it runs under. id is
// declared here rather than derived by constructing the analyzer and calling
// ID(): one builder (the commit emoji analyzer) does real I/O and returns nil
// on failure, which would silently drop it from any id-collecting code that
// had to construct first and ask second.
type analyzerSpec struct {
	id     string
	build  analyzerBuilder
	scopes []Scope
}

// staticScopes is every scope that runs the static/AST analyzers, the
// rule-injection stage, and the output stage: they stay useful with only a
// commit message (ScopeCommit) or only changed files (ScopeChanged), not
// just a full project context.
var staticScopes = []Scope{ScopeFull, ScopeCommit, ScopeChanged}

// archScopes is every scope that runs the architecture analyzers: they need
// a project or changed-file source set, so a bare commit-message check
// (ScopeCommit) skips them.
var archScopes = []Scope{ScopeFull, ScopeChanged}

// delegationScopes is every scope that pays the cost of spawning an external
// tool (golangci-lint, gopls): only a full run does.
var delegationScopes = []Scope{ScopeFull}

// registerForScope registers only the analyzers relevant to the given scope.
// A spec whose build returned nil (a build-time construction failure it
// already logged) is skipped rather than registered. repo may be nil when
// no git repository is available.
func (r *StageRunner) registerForScope(index *knowledge.Index, repo vcs.Repository, scope Scope) {
	langs := core.NewLanguageRegistry()
	langs.Register(golang.New())

	for _, spec := range analyzerSpecs {
		if !slices.Contains(spec.scopes, scope) {
			continue
		}
		if a := spec.build(index, langs, repo); a != nil {
			r.Register(a)
		}
	}
}

// RegisteredAnalyzerIDs returns the id of every analyzer declared in
// analyzerSpecs, across every scope. It reads the spec table directly rather
// than constructing each analyzer and calling ID(): one builder (the commit
// emoji analyzer) does real I/O and can return nil, which would silently drop
// that id from any collection built by constructing first and asking second.
func RegisteredAnalyzerIDs() []string {
	ids := make([]string, len(analyzerSpecs))
	for i, spec := range analyzerSpecs {
		ids[i] = spec.id
	}
	return ids
}
