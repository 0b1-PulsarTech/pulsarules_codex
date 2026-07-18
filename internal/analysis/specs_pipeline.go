package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineAnalyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewPackageBoundaryAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewImportCycleAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return delegation.NewGolangcilintAnalyzer("golangci-lint")
			},
			scopes: delegationScopes(),
		},
		{
			build: func(
				index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return ruleinjection.NewAnalyzer(index)
			},
			scopes: staticScopes(),
		},
		staticSpec(output.NewAnalyzer),
	}
}
