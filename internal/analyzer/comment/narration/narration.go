package narration

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string             { return "history-narration" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(analyzer.ID(), finding.SeverityWarning, finding.CatSyntax)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), language.SourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && language.ForExtension(fc.Extension).ID != ""
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			source := commentSource{
				lang:   language.ForExtension(fc.Extension),
				parsed: analyzer.cachedGo(ctx, fc),
			}
			return scanFile(src, fc, reporter, source)
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

func commentBlocks(
	src []byte, fc pass.FileChange, lang language.Language, parsed parsedGo,
) [][]segment {
	if parsed.file != nil {
		return blocks(fileSegments(parsed.file, parsed.fset))
	}
	if fc.Extension == ".go" {
		if segments, ok := goSegments(src); ok {
			return blocks(segments)
		}
	}
	return blocks(scanSegments(src, lang))
}

type commentSource struct {
	lang   language.Language
	parsed parsedGo
}

func scanFile(
	src []byte, fc pass.FileChange, reporter finding.Reporter, source commentSource,
) []finding.Finding {
	var findings []finding.Finding
	phrases := narrationPhrases()
	for _, block := range commentBlocks(src, fc, source.lang, source.parsed) {
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
