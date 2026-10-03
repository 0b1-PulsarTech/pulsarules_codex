package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/simplificationpath"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/textmarkers"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
)

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
