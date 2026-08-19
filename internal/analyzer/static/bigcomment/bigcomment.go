package bigcomment

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// why: 20 let every comment in this repo pass, so the rule "explain why, do not
// narrate what" had no machine behind it. Measured cost of the bar: 35 blocks
// at 5, 67 at 4. Five keeps a rationale that needs a sentence, refuses a
// paragraph.
const defaultMaxCommentLines = 5

var bigCommentReporter = core.NewReporter("big-comment", core.SeverityWarning, core.CatSyntax)

// Analyzer reports comment blocks exceeding a line threshold.
type Analyzer struct {
	langs    *core.LanguageRegistry
	maxLines int
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs, maxLines: defaultMaxCommentLines}
}

func (a *Analyzer) ID() string          { return "big-comment" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil {
		return nil
	}
	ml := a.resolveMaxLines(ctx)
	reporter := bigCommentReporter.Resolved(ctx)
	eligible := func(fc core.FileChange) bool { return a.langs.Lookup(fc.Extension) != nil }
	return core.EachChangedFile(ctx, eligible, func(fc core.FileChange, src []byte) []core.Finding {
		return scanFile(src, fc, a.langs.Lookup(fc.Extension), ml, reporter)
	})
}

// scanFile reads src line by line, tracking consecutive comment lines, and
// emits a finding when a comment block exceeds maxLines. Comment blocks that
// start with a generated-file marker or build-constraint tag are exempt.
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
