package governance

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/golang"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type analyzerBuilder func(
	index *knowledge.Index,
	langs *pass.LanguageRegistry,
	repo vcs.Repository,
) pass.Analyzer

type analyzerSpec struct {
	build  analyzerBuilder
	scopes []Scope
}

func staticScopes() []Scope {
	return []Scope{ScopeFull, ScopeCommit, ScopeChanged}
}

func archScopes() []Scope {
	return []Scope{ScopeFull, ScopeChanged}
}

func (r *StageRunner) registerForScope(index *knowledge.Index, repo vcs.Repository, scope Scope) {
	langs := pass.NewLanguageRegistry()
	langs.Register(golang.New())

	for _, spec := range analyzerSpecs() {
		if !slices.Contains(spec.scopes, scope) {
			continue
		}
		if a := spec.build(index, langs, repo); a != nil {
			r.Register(a)
		}
	}
}
