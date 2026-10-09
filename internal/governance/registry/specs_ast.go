package registry

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/comment/testdoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntaxreport"
)

func syntaxReportSpec(language syntax.Language) analyzerSpec {
	return staticSpec(string(language)+"-syntax", func() *syntaxreport.Analyzer {
		return syntaxreport.NewAnalyzer(language)
	})
}

func astSpecs() []analyzerSpec {
	return []analyzerSpec{
		syntaxReportSpec(syntax.LangGo),
		syntaxReportSpec(syntax.LangDart),
		syntaxReportSpec(syntax.LangTypeScript),
		syntaxReportSpec(syntax.LangJavaScript),
		syntaxReportSpec(syntax.LangSvelte),
		staticSpec("control-flow", controlflow.NewAnalyzer),
		staticSpec("shadowing", shadowing.NewAnalyzer),
		staticSpec("complexity", complexity.NewAnalyzer),
		staticSpec("named-returns", namedreturns.NewAnalyzer),
		staticSpec("test-comment", testdoc.NewAnalyzer),
		staticSpec("single-use-declaration", singleuse.NewAnalyzer),
	}
}
