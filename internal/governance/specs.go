package governance

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/simplification"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return append([]analyzerSpec{
		staticSpec("file-size", filesize.NewAnalyzer),
		staticSpec("no-em-dash", noemdash.NewAnalyzer),
		staticSpec("import-groups", importgroup.NewAnalyzer),
		staticSpec("naming", identifier.NewAnalyzer),
		langSpec("top-of-file", banner.NewAnalyzer),
		langSpec("big-comment", blocksize.NewAnalyzer),
		staticSpec("simplification-path", simplification.NewAnalyzer),
		{
			id: "commit-lint",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				catalog, err := emoji.NewCatalog()
				if err != nil {
					// simplification: analyzerBuilder returns only core.Analyzer; log
					// once and skip registering rather than widen that contract.
					// Upgrade path: give analyzerBuilder an error return.
					slog.Error("build commit emoji analyzer", slog.String("error", err.Error()))
					return nil
				}
				return message.NewAnalyzer(catalog)
			},
			scopes: staticScopes(),
		},
		{
			id: "commit-move-purity",
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, repo vcs.Repository,
			) pass.Analyzer {
				return movepurity.NewAnalyzer(repo)
			},
			scopes: staticScopes(),
		},
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
	}, pipelineAnalyzerSpecs()...)
}

func staticSpec[Built pass.Analyzer](id string, ctor func() Built) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository) pass.Analyzer {
			return ctor()
		},
		scopes: staticScopes(),
	}
}

func langSpec[Built pass.Analyzer](
	id string, ctor func(*pass.LanguageRegistry) Built,
) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(
			_ *knowledge.Index, langs *pass.LanguageRegistry, _ vcs.Repository,
		) pass.Analyzer {
			return ctor(langs)
		},
		scopes: staticScopes(),
	}
}
