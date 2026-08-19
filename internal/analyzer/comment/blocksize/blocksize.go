package blocksize

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const defaultMaxCommentLines = 5

type Analyzer struct {
	langs    *pass.LanguageRegistry
	maxLines int
	reporter pass.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs:    langs,
		maxLines: defaultMaxCommentLines,
		reporter: pass.NewReporter("big-comment", pass.SeverityWarning, pass.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "big-comment" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }
func (a *Analyzer) Needs() pass.Requirements {
	return pass.Requirements{}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if a.langs == nil || ctx.Sources == nil {
		return nil
	}
	ml := a.resolveMaxLines(ctx)
	var findings []pass.Finding
	for _, fc := range ctx.ChangedFiles {
		lang := a.langs.Lookup(fc.Extension)
		if lang == nil {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, scanFile(src, fc, lang, ml, a.reporter)...)
	}
	return findings
}

func scanFile(
	src []byte, fc pass.FileChange, lang pass.Language, maxLines int, reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	blockStart := 0
	blockLines := 0
	lineNum := 0
	exempt := false
	for line := range strings.SplitSeq(string(src), "\n") {
		lineNum++
		if lang.IsCommentLine(line) {
			if blockLines == 0 {
				blockStart = lineNum
				exempt = isGeneratedHeader(line) || isBuildTag(line)
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

func isGeneratedHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "// Code generated")
}

func isBuildTag(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//go:build") || strings.HasPrefix(trimmed, "// +build")
}

func (a *Analyzer) resolveMaxLines(ctx *pass.Context) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
