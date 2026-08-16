package analysis

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/vacuousdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/simplificationpath"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/textmarkers"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return append([]analyzerSpec{
		staticSpec("file-size", filesize.NewAnalyzer),
		staticSpec("text-markers", textmarkers.NewTextAnalyzer),
		staticSpec("typographic-markers", textmarkers.NewTypographicAnalyzer),
		staticSpec("import-groups", imports.NewAnalyzer),
		staticSpec("naming", naming.NewAnalyzer),
		langSpec("top-of-file", topoffile.NewAnalyzer),
		langSpec("big-comment", bigcomment.NewAnalyzer),
		staticSpec("simplification-path", simplificationpath.NewAnalyzer),
		{
			id: "commit-lint",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				catalog, err := emoji.NewCatalog()
				if err != nil {
					// simplification: analyzerBuilder returns only core.Analyzer; log
					// once and skip registering rather than widen that contract.
					// Upgrade path: give analyzerBuilder an error return.
					slog.Error("build commit emoji analyzer", slog.String("error", err.Error()))
					return nil
				}
				return commit.NewAnalyzer(catalog)
			},
			scopes: staticScopes(),
		},
		{
			id: "commit-move-purity",
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository,
			) core.Analyzer {
				return movepurity.NewAnalyzer(repo)
			},
			scopes: staticScopes(),
		},
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
		staticSpec("vacuous-doc", vacuousdoc.NewAnalyzer),
	}, pipelineAnalyzerSpecs()...)
}

func staticSpec[Built core.Analyzer](id string, ctor func() Built) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return ctor()
		},
		scopes: staticScopes(),
	}
}

func langSpec[Built core.Analyzer](
	id string, ctor func(*core.LanguageRegistry) Built,
) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(
			_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository,
		) core.Analyzer {
			return ctor(langs)
		},
		scopes: staticScopes(),
	}
}
