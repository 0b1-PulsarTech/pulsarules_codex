package registry

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/branch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/concurrency/clock"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/concurrency/loopexit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/security/pproflabel"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/rulesummary"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func pipelineSpecs() []analyzerSpec {
	return slices.Concat([]analyzerSpec{
		staticSpec("time-discipline", clock.NewAnalyzer),
		staticSpec("context-discipline", loopexit.NewAnalyzer),
		staticSpec("pprof-labels", pproflabel.NewAnalyzer),
		{
			// Full-only gate, which is where the pre-push hook runs. A branch
			// name is not a property of a staged change, so gating a commit on it
			// would block work that a rename before pushing fixes.
			id: "branch-name",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, repo vcs.Repository,
			) pass.Analyzer {
				return branch.NewAnalyzer(repo)
			},
			gate: GateFullOnly,
		},
		gatedSpec("arch-boundary", GateArch, arch.NewPackageBoundaryAnalyzer),
		gatedSpec("import-cycle", GateArch, arch.NewImportCycleAnalyzer),
	}, delegationSpecs(), []analyzerSpec{
		{
			id: "rule-injection",
			build: func(
				index *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return rulesummary.NewAnalyzer(index)
			},
			gate:     GateStatic,
			internal: true,
		},
		{
			id: "output",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return output.NewAnalyzer()
			},
			gate:     GateStatic,
			internal: true,
		},
	})
}

func delegationSpecs() []analyzerSpec {
	return []analyzerSpec{
		{
			id: "golangci-lint",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return golangcilint.NewAnalyzer("golangci-lint")
			},
			gate: GateFullOnly,
		},
		{
			id: command.ID,
			build: func(
				index *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return command.NewAnalyzer(index)
			},
			gate:     GateFullOnly,
			internal: true,
		},
	}
}

func DelegatedFormats() []string {
	return command.Formats()
}
