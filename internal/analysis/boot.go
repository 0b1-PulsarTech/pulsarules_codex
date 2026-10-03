package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (runner *StageRunner) registerForScope(
	index *knowledge.Index,
	repo vcs.Repository,
	scope Scope,
) {
	langs := core.NewLanguageRegistry()
	langspec.RegisterAll(langs)

	for _, analyzer := range registry.Analyzers(index, langs, repo, gatesFor(scope)) {
		runner.Register(analyzer)
	}
}

func gatesFor(scope Scope) []registry.Gate {
	switch scope {
	case ScopeCommit:
		// The pre-commit and commit-msg hooks run it, so only the fast static gate may.
		return []registry.Gate{registry.GateStatic}
	case ScopeChanged:
		// golangci-lint needs the whole project; branch-name guards a push.
		return []registry.Gate{registry.GateStatic, registry.GateArch}
	default:
		return []registry.Gate{registry.GateStatic, registry.GateArch, registry.GateFullOnly}
	}
}
