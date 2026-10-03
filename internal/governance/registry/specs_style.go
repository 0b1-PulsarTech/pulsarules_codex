package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/bodyheader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/doc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/narration"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/packagevar"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/filecount"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/resultcount"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/variadicindex"
)

func styleSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("doc-comment", doc.NewAnalyzer),
		staticSpec("body-header-comment", bodyheader.NewAnalyzer),
		langSpec("history-narration", narration.NewAnalyzer),
		staticSpec("variadic-index", variadicindex.NewAnalyzer),
		staticSpec("too-many-results", resultcount.NewAnalyzer),
		staticSpec("package-level-var", packagevar.NewAnalyzer),
		staticSpec("package-file-count", filecount.NewAnalyzer),
	}
}
