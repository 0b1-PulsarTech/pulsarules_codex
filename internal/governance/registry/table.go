package registry

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/character"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/bodyheader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/doc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/narration"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/simplification"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/testdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/branch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/concurrency/clock"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/concurrency/loopexit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/interfacegroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/packagevar"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filecount"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/singleletter"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/security/pproflabel"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/resultcount"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/variadicindex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntaxreport"
)

func specs() []entry {
	// Run order is table order; ordering and dedupe of the results is a pipeline transform.
	return slices.Concat(contentSpecs(), syntaxSpecs(), styleSpecs(), []entry{
		plain(arch.DescribeBoundary(), GateArch, arch.NewPackageBoundaryAnalyzer),
		plain(arch.DescribeCycle(), GateArch, arch.NewImportCycleAnalyzer),
	})
}

func contentSpecs() []entry {
	return []entry{
		plain(filesize.Describe(), GateStatic, filesize.NewAnalyzer),
		plain(character.DescribeText(), GateStatic, character.NewTextAnalyzer),
		plain(character.DescribeTypographic(), GateStatic, character.NewTypographicAnalyzer),
		plain(importgroup.Describe(), GateStatic, importgroup.NewAnalyzer),
		plain(identifier.Describe(), GateStatic, identifier.NewAnalyzer),
		plain(banner.Describe(), GateStatic, banner.NewAnalyzer),
		plain(blocksize.Describe(), GateStatic, blocksize.NewAnalyzer),
		plain(simplification.Describe(), GateStatic, simplification.NewAnalyzer),
		viaDeps(message.Describe(), GateStatic, func(deps Deps) pass.Analyzer {
			return message.NewAnalyzer(deps.Emoji)
		}),
		viaDeps(movepurity.Describe(), GateStatic, func(deps Deps) pass.Analyzer {
			return movepurity.NewAnalyzer(deps.Repo)
		}),
		plain(narration.Describe(), GateStatic, narration.NewAnalyzer),
		// Full-only gate, which is where the pre-push hook runs. A branch name is not a
		// property of a staged change, so gating a commit on it would block work that a
		// rename before pushing fixes.
		viaDeps(branch.Describe(), GateFullOnly, func(deps Deps) pass.Analyzer {
			return branch.NewAnalyzer(deps.Repo)
		}),
		plain(golangcilint.Describe(), GateFullOnly, func() *golangcilint.Analyzer {
			return golangcilint.NewAnalyzer("golangci-lint")
		}),
		hide(viaDeps(command.Describe(), GateFullOnly, func(deps Deps) pass.Analyzer {
			return command.NewAnalyzer(deps.Index)
		})),
	}
}

func syntaxSpecs() []entry {
	languages := syntaxreport.Languages()
	specs := make([]entry, 0, len(languages))
	for _, lang := range languages {
		specs = append(specs, plain(syntaxreport.Describe(lang), GateStatic,
			func() *syntaxreport.Analyzer { return syntaxreport.NewAnalyzer(lang) }))
	}
	return specs
}

func styleSpecs() []entry {
	return []entry{
		plain(controlflow.Describe(), GateStatic, controlflow.NewAnalyzer),
		plain(shadowing.Describe(), GateStatic, shadowing.NewAnalyzer),
		plain(complexity.Describe(), GateStatic, complexity.NewAnalyzer),
		plain(namedreturns.Describe(), GateStatic, namedreturns.NewAnalyzer),
		plain(testdoc.Describe(), GateStatic, testdoc.NewAnalyzer),
		plain(singleuse.Describe(), GateStatic, singleuse.NewAnalyzer),
		plain(doc.Describe(), GateStatic, doc.NewAnalyzer),
		plain(bodyheader.Describe(), GateStatic, bodyheader.NewAnalyzer),
		plain(singleletter.Describe(), GateStatic, singleletter.NewAnalyzer),
		plain(variadicindex.Describe(), GateStatic, variadicindex.NewAnalyzer),
		plain(resultcount.Describe(), GateStatic, resultcount.NewAnalyzer),
		plain(interfacegroup.Describe(), GateStatic, interfacegroup.NewAnalyzer),
		plain(packagevar.Describe(), GateStatic, packagevar.NewAnalyzer),
		plain(filecount.Describe(), GateStatic, filecount.NewAnalyzer),
		plain(clock.Describe(), GateStatic, clock.NewAnalyzer),
		plain(loopexit.Describe(), GateStatic, loopexit.NewAnalyzer),
		plain(pproflabel.Describe(), GateStatic, pproflabel.NewAnalyzer),
	}
}
