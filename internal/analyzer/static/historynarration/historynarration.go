package historynarration

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type Analyzer struct {
	langs *core.LanguageRegistry
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs}
}

func (analyzer *Analyzer) ID() string          { return "history-narration" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatSyntax)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if analyzer.langs == nil {
		return nil
	}
	reporter := analyzer.reporter().Resolved(ctx)
	extFilter := core.ExtensionFilter(ctx, analyzer.ID(), core.DefaultSourceExtensions())
	eligible := func(fc core.FileChange) bool {
		return extFilter(fc) && analyzer.langs.Lookup(fc.Extension) != nil
	}
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			syntax := analyzer.langs.Lookup(fc.Extension).CommentSyntax()
			source := commentSource{syntax: syntax, parsed: analyzer.cachedGo(ctx, fc)}
			return scanFile(src, fc, reporter, source)
		},
	)
}

type parsedGo struct {
	file *ast.File
	fset *token.FileSet
}

func (analyzer *Analyzer) cachedGo(ctx *core.AnalysisContext, fc core.FileChange) parsedGo {
	if fc.Extension != ".go" || ctx.ASTCache == nil {
		return parsedGo{}
	}
	return parsedGo{file: ctx.ASTCache.Get(fc.Path), fset: ctx.ASTCache.FileSet()}
}

func commentBlocks(
	src []byte, fc core.FileChange, syntax lexical.CommentSyntax, parsed parsedGo,
) [][]segment {
	if parsed.file != nil {
		return blocks(fileSegments(parsed.file, parsed.fset))
	}
	if fc.Extension == ".go" {
		if segments, ok := goSegments(src); ok {
			return blocks(segments)
		}
	}
	return blocks(scanSegments(src, syntax, fc.Extension))
}

type commentSource struct {
	syntax lexical.CommentSyntax
	parsed parsedGo
}

func scanFile(
	src []byte, fc core.FileChange, reporter core.Reporter, source commentSource,
) []core.Finding {
	var findings []core.Finding
	phrases := narrationPhrases()
	for _, block := range commentBlocks(src, fc, source.syntax, source.parsed) {
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
