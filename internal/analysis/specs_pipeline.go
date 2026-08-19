package analysis

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/timediscipline"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/branchname"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineAnalyzerSpecs() []analyzerSpec {
	return slices.Concat([]analyzerSpec{
		staticSpec("time-discipline", timediscipline.NewAnalyzer),
		{
			id: "branch-name",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository,
			) core.Analyzer {
				return branchname.NewAnalyzer(repo)
			},
			scopes: []Scope{ScopeFull},
		},
		{
			id: "arch-boundary",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewPackageBoundaryAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			id: "import-cycle",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewImportCycleAnalyzer()
			},
			scopes: archScopes(),
		},
	}, delegationSpecs(), []analyzerSpec{
		{
			id: "rule-injection",
			build: func(
				index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return ruleinjection.NewAnalyzer(index)
			},
			scopes: staticScopes(),
		},
		staticSpec("output", output.NewAnalyzer),
	})
}

func delegationSpecs() []analyzerSpec {
	return []analyzerSpec{
		{
			id: "golangci-lint",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return delegation.NewGolangcilintAnalyzer("golangci-lint")
			},
			scopes: delegationScopes(),
		},
	}
}
