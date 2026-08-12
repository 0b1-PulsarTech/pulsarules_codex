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

// pipelineAnalyzerSpecs is the architecture, delegated-tool, and
// pipeline-housekeeping tail of analyzerSpecs (specs.go), split into this
// file only to keep specs.go under the file-size ceiling.
var pipelineAnalyzerSpecs = []analyzerSpec{
	{
		id: "arch-boundary",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return arch.NewPackageBoundaryAnalyzer()
		},
		scopes: archScopes,
	},
	{
		id: "import-cycle",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return arch.NewImportCycleAnalyzer()
		},
		scopes: archScopes,
	},
	{
		id: "golangci-lint",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return delegation.NewGolangcilintAnalyzer("golangci-lint")
		},
		scopes: delegationScopes,
	},
	{
		id: "gopls",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return delegation.NewGoplsAnalyzer()
		},
		scopes: delegationScopes,
	},
	{
		id: "rule-injection",
		build: func(index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return ruleinjection.NewAnalyzer(index)
		},
		scopes: staticScopes,
	},
	{
		id: "output",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return output.NewAnalyzer()
		},
		scopes: staticScopes,
	},
}
