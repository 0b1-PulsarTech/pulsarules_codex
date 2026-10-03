package blocksize

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type Analyzer struct {
	langs    *pass.LanguageRegistry
	maxLines int
	reporter pass.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	const defaultMaxCommentLines = 3
	return &Analyzer{
		langs: langs,
		// A kept comment justifies itself in one sentence: three lines fits it, wrapped;
		// a fourth is the first line of a paragraph.
		maxLines: defaultMaxCommentLines,
		reporter: pass.NewReporter("big-comment", pass.SeverityWarning, pass.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "big-comment" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if a.langs == nil {
		return nil
	}
	ml := a.resolveMaxLines(ctx)
	reporter := pass.ResolveReporter(a.reporter, ctx)
	extFilter := pass.ExtensionFilter(ctx, a.ID(), pass.DefaultSourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []pass.Finding {
			syntax := a.langs.Lookup(fc.Extension).CommentSyntax()
			return scanFile(src, fc, syntax, ml, reporter)
		},
	)
}

func (a *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
