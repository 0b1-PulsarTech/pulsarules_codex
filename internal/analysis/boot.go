package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (r *StageRunner) registerForScope(index *knowledge.Index, repo vcs.Repository, scope Scope) {
	langs := core.NewLanguageRegistry()
	langspec.RegisterAll(langs)

	for _, a := range Analyzers(index, langs, repo, gatesFor(scope)) {
		r.Register(a)
	}
}

func gatesFor(scope Scope) []Gate {
	switch scope {
	case ScopeCommit:
		// The pre-commit and commit-msg hooks run it, so only the fast static gate may.
		return []Gate{GateStatic}
	case ScopeChanged:
		// golangci-lint needs the whole project; branch-name guards a push.
		return []Gate{GateStatic, GateArch}
	default:
		return []Gate{GateStatic, GateArch, GateFullOnly}
	}
}
