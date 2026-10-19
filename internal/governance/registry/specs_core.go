package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/character"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/banner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/blocksize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/simplification"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
)

func coreSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("file-size", filesize.NewAnalyzer),
		staticSpec("text-markers", character.NewTextAnalyzer),
		staticSpec("typographic-markers", character.NewTypographicAnalyzer),
		staticSpec("import-groups", importgroup.NewAnalyzer),
		staticSpec("naming", identifier.NewAnalyzer),
		staticSpec("top-of-file", banner.NewAnalyzer),
		staticSpec("big-comment", blocksize.NewAnalyzer),
		staticSpec("simplification-path", simplification.NewAnalyzer),
	}
}
