package analysis

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/simplificationpath"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/textmarkers"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return slices.Concat(coreSpecs(), commitSpecs(), astSpecs(), pipelineAnalyzerSpecs())
}

func coreSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("file-size", filesize.NewAnalyzer),
		staticSpec("text-markers", textmarkers.NewTextAnalyzer),
		staticSpec("typographic-markers", textmarkers.NewTypographicAnalyzer),
		staticSpec("import-groups", imports.NewAnalyzer),
		staticSpec("naming", naming.NewAnalyzer),
		langSpec("top-of-file", topoffile.NewAnalyzer),
		langSpec("big-comment", bigcomment.NewAnalyzer),
		langSpec("simplification-path", simplificationpath.NewAnalyzer),
	}
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
