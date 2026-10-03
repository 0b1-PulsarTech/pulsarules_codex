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

func (a *Analyzer) ID() string          { return "big-comment" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil {
		return nil
	}
	ml := a.resolveMaxLines(ctx)
	reporter := a.reporter.Resolved(ctx)
	extFilter := core.ExtensionFilter(ctx, a.ID(), core.DefaultSourceExtensions())
	eligible := func(fc core.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			syntax := a.langs.Lookup(fc.Extension).CommentSyntax()
			return scanFile(src, fc, syntax, ml, reporter)
		},
	)
}

func (a *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
