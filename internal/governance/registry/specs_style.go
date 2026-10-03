package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/bodyheader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/doc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/narration"
)

func styleSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("doc-comment", doc.NewAnalyzer),
		staticSpec("body-header-comment", bodyheader.NewAnalyzer),
		langSpec("history-narration", narration.NewAnalyzer),
	}
}
