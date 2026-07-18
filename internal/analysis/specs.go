package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
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
		staticSpec(controlflow.NewAnalyzer),
		staticSpec(shadowing.NewAnalyzer),
		staticSpec(complexity.NewAnalyzer),
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewPackageBoundaryAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			build: func(
				_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return arch.NewImportCycleAnalyzer()
			},
			scopes: archScopes(),
		},
		{
			build: func(
				index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository,
			) core.Analyzer {
				return ruleinjection.NewAnalyzer(index)
			},
			scopes: staticScopes(),
		},
		staticSpec(output.NewAnalyzer),
	}
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
