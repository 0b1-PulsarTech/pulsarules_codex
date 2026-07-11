package bigcomment

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const defaultMaxCommentLines = 20

var bigCommentReporter = core.NewReporter("big-comment", core.SeverityWarning, core.CatSyntax)

// Analyzer reports comment blocks exceeding a line threshold.
type Analyzer struct {
	langs    *core.LanguageRegistry
	maxLines int
}

// NewAnalyzer creates an Analyzer with the default
// threshold and the given language registry.
func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs, maxLines: defaultMaxCommentLines}
}

func (a *Analyzer) ID() string   { return "big-comment" }
func (a *Analyzer) Name() string { return "Big comment" }
func (a *Analyzer) Description() string {
	return "Reports comment blocks that exceed the configured line count"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageStatic }
func (a *Analyzer) Category() core.Category { return core.CatSyntax }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil || ctx.Sources == nil {
		return nil
	}
	ml := a.resolveMaxLines(ctx)
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		lang := a.langs.Lookup(fc.Extension)
		if lang == nil {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, scanFile(src, fc, lang, ml)...)
	}
	return findings
}

// scanFile reads src line by line, tracking consecutive comment lines, and
// emits a finding when a comment block exceeds maxLines. Comment blocks that
// start with a generated-file marker or build-constraint tag are exempt.
func scanFile(src []byte, fc core.FileChange, lang core.Language, maxLines int) []core.Finding {
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
				exempt = isGeneratedHeader(line) || isBuildTag(line)
			}
			blockLines++
			continue
		}
		if blockLines > maxLines && !exempt {
			findings = append(findings, bigCommentReporter.At(
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
		findings = append(findings, bigCommentReporter.At(
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

func (a *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}
