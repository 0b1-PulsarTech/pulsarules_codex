package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/rulesummary"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
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
		staticSpec(controlflow.NewAnalyzer),
		staticSpec(shadowing.NewAnalyzer),
		staticSpec(complexity.NewAnalyzer),
		{
			build: func(
				index *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository,
			) pass.Analyzer {
				return rulesummary.NewAnalyzer(index)
			},
			scopes: staticScopes(),
		},
		staticSpec(output.NewAnalyzer),
	}
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
