package blocksize

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	langs    *pass.LanguageRegistry
	maxLines int
	reporter finding.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	const defaultMaxCommentLines = 3
	return &Analyzer{
		langs: langs,
		// A kept comment justifies itself in one sentence: three lines fits it, wrapped;
		// a fourth is the first line of a paragraph.
		maxLines: defaultMaxCommentLines,
		reporter: finding.NewReporter("big-comment", finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string             { return "big-comment" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if analyzer.langs == nil {
		return nil
	}
	ml := analyzer.resolveMaxLines(ctx)
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), pass.DefaultSourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && analyzer.langs.Lookup(fc.Extension) != nil
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			syntax := analyzer.langs.Lookup(fc.Extension).CommentSyntax()
			return scanFile(src, fc, syntax, limit{maxLines: ml, reporter: reporter})
		},
	)
}

func (analyzer *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(analyzer.ID()).Int("max_lines", analyzer.maxLines)
}
