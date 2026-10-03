package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/bodyheadercomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/doccomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/packagefilecount"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/packagelevelvar"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/toomanyresults"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/ungroupedinterfaces"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/variadicindex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/historynarration"
)

func styleSpecs() []analyzerSpec {
	return []analyzerSpec{
		staticSpec("doc-comment", doccomment.NewAnalyzer),
		staticSpec("body-header-comment", bodyheadercomment.NewAnalyzer),
		langSpec("history-narration", historynarration.NewAnalyzer),
		staticSpec("variadic-index", variadicindex.NewAnalyzer),
		staticSpec("too-many-results", toomanyresults.NewAnalyzer),
		staticSpec("ungrouped-interfaces", ungroupedinterfaces.NewAnalyzer),
		staticSpec("package-level-var", packagelevelvar.NewAnalyzer),
		staticSpec("package-file-count", packagefilecount.NewAnalyzer),
	}
}
