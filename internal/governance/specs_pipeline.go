package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/rulesummary"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineAnalyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
		{
			id: "arch-boundary",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return arch.NewPackageBoundaryAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			id: "import-cycle",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return arch.NewImportCycleAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			id: "golangci-lint",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return golangcilint.NewAnalyzer("golangci-lint")
			},
			scopes: delegationScopes(),
		},
		{
			id: "rule-injection",
			build: func(
				index *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return rulesummary.NewAnalyzer(index)
			},
			scopes: staticScopes(),
		},
		staticSpec("output", output.NewAnalyzer),
	}
}
