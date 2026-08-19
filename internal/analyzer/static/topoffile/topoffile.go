package topoffile

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	langs    *core.LanguageRegistry
	reporter core.Reporter
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs:    langs,
		reporter: core.NewReporter("top-of-file", core.SeverityError, core.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil || ctx.Sources == nil {
		return nil
	}
	reporter := a.reporter.Resolved(ctx)
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
		line, lineNum, ok := firstNonBlank(src)
		if !ok {
			continue
		}
		if core.IsExemptHeaderLine(line) {
			continue
		}
		if lang.IsCommentLine(line) && !lang.IsPackageDeclaration(line) {
			findings = append(findings, reporter.At(
				fc.Path,
				lineNum,
				"file starts with a comment before the package declaration",
				"remove the comment; no package docstrings",
			))
		}
	}
	return findings
}

func firstNonBlank(src []byte) (string, int, bool) {
	lineNum := 0
	for line := range strings.SplitSeq(string(src), "\n") {
		lineNum++
		if strings.TrimSpace(line) == "" {
			continue
		}
		return line, lineNum, true
	}
	return "", 0, false
}
