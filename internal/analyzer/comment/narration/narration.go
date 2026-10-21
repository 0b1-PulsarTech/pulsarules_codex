package narration

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const ID = "history-narration"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(ID, finding.SeverityWarning, finding.CatSyntax)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := analyzer.reporter()
	extFilter := pass.ExtensionFilter(ctx, ID, language.SourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && language.ForExtension(fc.Extension).ID != ""
	}
	return ctx.EachSource(
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			return scanFile(src, fc, reporter, analyzer.cachedGo(ctx, fc))
		},
	)
}

type parsedGo struct {
	file *ast.File
	fset *token.FileSet
}

func (analyzer *Analyzer) cachedGo(ctx *pass.Context, fc pass.FileChange) parsedGo {
	if fc.Extension != ".go" || ctx.ASTCache == nil {
		return parsedGo{}
	}
	return parsedGo{file: ctx.ASTCache.Get(fc.Path), fset: ctx.ASTCache.FileSet()}
}

func commentBlocks(src []byte, fc pass.FileChange, parsed parsedGo) [][]segment {
	if parsed.file != nil {
		return blocks(fileSegments(parsed.file, parsed.fset))
	}
	return blocks(scanSegments(string(src), fc.Extension))
}

func scanFile(
	src []byte, fc pass.FileChange, reporter finding.Reporter, parsed parsedGo,
) []finding.Finding {
	var findings []finding.Finding
	phrases := narrationPhrases()
	for _, block := range commentBlocks(src, fc, parsed) {
		for _, found := range findHits(wordsOf(block), phrases) {
			findings = append(findings, reporter.At(
				fc.Path,
				found.line,
				"comment narrates history: \""+found.phrase+"\"",
				"describe what the code does now; the change history belongs in the "+
					"commit message",
			))
		}
	}
	return findings
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "comments",
		Params: []string{pass.ParamExtensions},
	}
}
