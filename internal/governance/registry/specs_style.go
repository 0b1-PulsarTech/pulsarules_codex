package registry

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/doc"

func styleSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("doc-comment", doc.NewAnalyzer),
	}
}
