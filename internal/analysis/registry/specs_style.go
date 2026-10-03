package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/bodyheadercomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/doccomment"
)

func styleSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("doc-comment", doccomment.NewAnalyzer),
		staticSpec("body-header-comment", bodyheadercomment.NewAnalyzer),
	}
}
