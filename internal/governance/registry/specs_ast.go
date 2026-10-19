package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/testdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntaxreport"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func syntaxReportSpec(language language.ID) analyzerSpec {
	return staticSpec(string(language)+"-syntax", func() *syntaxreport.Analyzer {
		return syntaxreport.NewAnalyzer(language)
	})
}

func astSpecs() []analyzerSpec {
	var specs []analyzerSpec
	for _, id := range syntaxreport.Languages() {
		specs = append(specs, syntaxReportSpec(id))
	}
	return append(specs, []analyzerSpec{
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
		staticSpec("test-comment", testdoc.NewAnalyzer),
		staticSpec("single-use-declaration", singleuse.NewAnalyzer),
	}...)
}
