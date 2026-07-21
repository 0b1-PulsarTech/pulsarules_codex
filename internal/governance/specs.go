package governance

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
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
				_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return importgroup.NewAnalyzer("github.com/0b1-PulsarTech/pulsarules_codex")
			},
			scopes: staticScopes(),
		},
		staticSpec(identifier.NewAnalyzer),
		langSpec(banner.NewAnalyzer),
		langSpec(blocksize.NewAnalyzer),
		{
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
			build: func(
				_ *knowledge.Index, _ *pass.LanguageRegistry, repo vcs.Repository,
			) pass.Analyzer {
				return movepurity.NewAnalyzer(repo)
			},
			scopes: staticScopes(),
		},
		staticSpec(controlflow.NewAnalyzer),
		staticSpec(shadowing.NewAnalyzer),
		staticSpec(complexity.NewAnalyzer),
	}, pipelineAnalyzerSpecs()...)
}

func staticSpec[Built pass.Analyzer](ctor func() Built) analyzerSpec {
	return analyzerSpec{
		build: func(_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository) pass.Analyzer {
			return ctor()
		},
		scopes: staticScopes(),
	}
}

func langSpec[Built pass.Analyzer](ctor func(*pass.LanguageRegistry) Built) analyzerSpec {
	return analyzerSpec{
		build: func(
			_ *knowledge.Index, langs *pass.LanguageRegistry, _ vcs.Repository,
		) pass.Analyzer {
			return ctor(langs)
		},
		scopes: staticScopes(),
	}
}
