package blocksize

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const ID = "big-comment"

const paramMaxLines = "max_lines"

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
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	ml := analyzer.resolveMaxLines(ctx)
	reporter := analyzer.reporter
	extFilter := pass.ExtensionFilter(ctx, ID, language.SourceExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && language.ForExtension(fc.Extension).ID != ""
	}
	return ctx.EachSource(
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			syntax := language.ForExtension(fc.Extension).Comments
			return scanFile(src, fc, syntax, limit{maxLines: ml, reporter: reporter})
		},
	)
}

func (analyzer *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(ID).Int(paramMaxLines, analyzer.maxLines)
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "comments",
		Params: []string{paramMaxLines, pass.ParamExtensions},
	}
}
