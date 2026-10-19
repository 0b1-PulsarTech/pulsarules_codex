package blocksize

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Analyzer struct {
	maxLines int
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxCommentLines = 3
	return &Analyzer{
		// A kept comment justifies itself in one sentence: three lines fits it, wrapped;
		// a fourth is the first line of a paragraph.
		maxLines: defaultMaxCommentLines,
		reporter: finding.NewReporter("big-comment", finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string             { return "big-comment" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	ml := analyzer.resolveMaxLines(ctx)
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), language.SourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && language.ForExtension(fc.Extension).ID != ""
	}
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			syntax := language.ForExtension(fc.Extension).Comments
			return scanFile(src, fc, syntax, limit{maxLines: ml, reporter: reporter})
		},
	)
}

func (analyzer *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(analyzer.ID()).Int("max_lines", analyzer.maxLines)
}
