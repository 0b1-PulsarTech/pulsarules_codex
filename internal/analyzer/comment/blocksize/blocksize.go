package blocksize

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

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

func scanFile(
	src []byte,
	fc pass.FileChange,
	syntax pass.CommentSyntax,
	maxLines int,
	reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	scanner := pass.NewCommentScanner(syntax)
	blockStart := 0
	blockLines := 0
	lineNum := 0
	exempt := false
	for line := range strings.SplitSeq(string(src), "\n") {
		lineNum++
		if scanner.IsCommentLine(line) {
			if pass.IsDirectiveHeaderLine(line) {
				continue
			}
			if blockLines == 0 {
				blockStart = lineNum
				// A simplification block states the ceiling and an upgrade path, which no
				// 3-line cap holds.
				exempt = pass.IsGeneratedMarkerLine(line) || pass.IsSimplificationMarker(line)
			}
			blockLines++
			continue
		}
		if blockLines > maxLines && !exempt {
			findings = append(findings, reporter.At(
				fc.Path,
				blockStart,
				"comment block is "+strconv.Itoa(blockLines)+" lines, max "+strconv.Itoa(maxLines),
				"split the comment or move the explanation into named functions/types",
			))
		}
		blockLines = 0
		exempt = false
	}
	if blockLines > maxLines && !exempt {
		findings = append(findings, reporter.At(
			fc.Path,
			blockStart,
			"comment block is "+strconv.Itoa(blockLines)+" lines, max "+strconv.Itoa(maxLines),
			"split the comment or move the explanation into named functions/types",
		))
	}
	return findings
}

func (a *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
