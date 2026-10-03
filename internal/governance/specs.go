package governance

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/character"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/simplification"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func analyzerSpecs() []analyzerSpec {
	return slices.Concat(coreSpecs(), commitSpecs(), astSpecs(), pipelineAnalyzerSpecs())
}

func coreSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("file-size", filesize.NewAnalyzer),
		staticSpec("text-markers", character.NewTextAnalyzer),
		staticSpec("typographic-markers", character.NewTypographicAnalyzer),
		staticSpec("import-groups", importgroup.NewAnalyzer),
		staticSpec("naming", identifier.NewAnalyzer),
		langSpec("top-of-file", banner.NewAnalyzer),
		langSpec("big-comment", blocksize.NewAnalyzer),
		langSpec("simplification-path", simplification.NewAnalyzer),
	}
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
