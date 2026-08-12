package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/timediscipline"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineAnalyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("time-discipline", timediscipline.NewAnalyzer),
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
		{
			id: "golangci-lint",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return delegation.NewGolangcilintAnalyzer("golangci-lint")
			},
			scopes: delegationScopes(),
		},
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
	}
}
