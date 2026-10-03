package bigcomment

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"

type Analyzer struct {
	langs    *core.LanguageRegistry
	maxLines int
	reporter core.Reporter
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	const defaultMaxCommentLines = 3
	return &Analyzer{
		langs: langs,
		// A kept comment justifies itself in one sentence: three lines fits it, wrapped;
		// a fourth is the first line of a paragraph.
		maxLines: defaultMaxCommentLines,
		reporter: core.NewReporter("big-comment", core.SeverityWarning, core.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string          { return "big-comment" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if analyzer.langs == nil {
		return nil
	}
	ml := analyzer.resolveMaxLines(ctx)
	reporter := analyzer.reporter.Resolved(ctx)
	extFilter := core.ExtensionFilter(ctx, analyzer.ID(), core.DefaultSourceExtensions())
	eligible := func(fc core.FileChange) bool {
		return extFilter(fc) && analyzer.langs.Lookup(fc.Extension) != nil
	}
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			syntax := analyzer.langs.Lookup(fc.Extension).CommentSyntax()
			return scanFile(src, fc, syntax, limit{maxLines: ml, reporter: reporter})
		},
	)
}

func (analyzer *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(analyzer.ID()).Int("max_lines", analyzer.maxLines)
}
