package governance

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type analyzerBuilder func(
	index *knowledge.Index,
	langs *pass.LanguageRegistry,
	repo vcs.Repository,
) pass.Analyzer

type analyzerSpec struct {
	id     string
	build  analyzerBuilder
	scopes []Scope
}

func staticScopes() []Scope {
	return []Scope{ScopeFull, ScopeCommit, ScopeChanged}
}

func archScopes() []Scope {
	return []Scope{ScopeFull, ScopeChanged}
}

func delegationScopes() []Scope {
	return []Scope{ScopeFull}
}

func (r *StageRunner) registerForScope(index *knowledge.Index, repo vcs.Repository, scope Scope) {
	langs := pass.NewLanguageRegistry()
	language.RegisterAll(langs)

	for _, spec := range analyzerSpecs() {
		if !slices.Contains(spec.scopes, scope) {
			continue
		}
		if a := spec.build(index, langs, repo); a != nil {
			r.Register(a)
		}
	}
}

func RegisteredAnalyzerIDs() []string {
	specs := analyzerSpecs()
	ids := make([]string, len(specs))
	for i, spec := range specs {
		ids[i] = spec.id
	}
	return ids
}
