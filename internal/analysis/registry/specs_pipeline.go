package registry

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/contextdiscipline"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/pproflabels"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/timediscipline"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/branchname"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineSpecs() []analyzerSpec {
	return slices.Concat([]analyzerSpec{
		staticSpec("time-discipline", timediscipline.NewAnalyzer),
		staticSpec("context-discipline", contextdiscipline.NewAnalyzer),
		staticSpec("pprof-labels", pproflabels.NewAnalyzer),
		{
			// Full-only gate, which is where the pre-push hook runs. A branch
			// name is not a property of a staged change, so gating a commit on it
			// would block work that a rename before pushing fixes.
			id: "branch-name",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository,
			) core.Analyzer {
				return branchname.NewAnalyzer(repo)
			},
			gate: GateFullOnly,
		},
		gatedSpec("arch-boundary", GateArch, arch.NewPackageBoundaryAnalyzer),
		gatedSpec("import-cycle", GateArch, arch.NewImportCycleAnalyzer),
	}, delegationSpecs(), []analyzerSpec{
		{
			id: "rule-injection",
			build: func(
				index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return ruleinjection.NewAnalyzer(index)
			},
			gate: GateStatic,
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
			gate: GateFullOnly,
		},
	}
}
