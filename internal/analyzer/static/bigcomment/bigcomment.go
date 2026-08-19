package bigcomment

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const defaultMaxCommentLines = 5

type Analyzer struct {
	langs    *core.LanguageRegistry
	maxLines int
	reporter core.Reporter
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs:    langs,
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
	eligible := func(fc core.FileChange) bool { return a.langs.Lookup(fc.Extension) != nil }
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			return scanFile(src, fc, a.langs.Lookup(fc.Extension), ml, reporter)
		},
	)
}

func scanFile(
	src []byte, fc core.FileChange, lang core.Language, maxLines int, reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	blockStart := 0
	blockLines := 0
	lineNum := 0
	exempt := false
	for line := range strings.SplitSeq(string(src), "\n") {
		lineNum++
		if lang.IsCommentLine(line) {
			if blockLines == 0 {
				blockStart = lineNum
				exempt = core.IsExemptHeaderLine(line)
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

func (a *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
