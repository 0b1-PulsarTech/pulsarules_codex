package analysis

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return append([]analyzerSpec{
		staticSpec(filesize.NewAnalyzer),
		staticSpec(noemdash.NewAnalyzer),
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return imports.NewAnalyzer("github.com/0b1-PulsarTech/pulsarules_codex")
			},
			scopes: staticScopes(),
		},
		staticSpec(naming.NewAnalyzer),
		langSpec(topoffile.NewAnalyzer),
		langSpec(bigcomment.NewAnalyzer),
		{
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
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository,
			) core.Analyzer {
				return movepurity.NewAnalyzer(repo)
			},
			scopes: staticScopes(),
		},
		staticSpec(controlflow.NewAnalyzer),
		staticSpec(shadowing.NewAnalyzer),
		staticSpec(complexity.NewAnalyzer),
		staticSpec(namedreturns.NewAnalyzer),
	}, pipelineAnalyzerSpecs()...)
}

func staticSpec[Built core.Analyzer](ctor func() Built) analyzerSpec {
	return analyzerSpec{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return ctor()
		},
		scopes: staticScopes(),
	}
}

func langSpec[Built core.Analyzer](ctor func(*core.LanguageRegistry) Built) analyzerSpec {
	return analyzerSpec{
		build: func(
			_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository,
		) core.Analyzer {
			return ctor(langs)
		},
		scopes: staticScopes(),
	}
}
