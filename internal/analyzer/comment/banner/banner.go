package banner

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	langs    *pass.LanguageRegistry
	reporter pass.Reporter
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{
		langs:    langs,
		reporter: pass.NewReporter("top-of-file", pass.SeverityError, pass.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if a.langs == nil || ctx.Sources == nil {
		return nil
	}
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
		line, lineNum, ok := firstNonBlank(src)
		if !ok {
			continue
		}
		if isExemptHeader(line) {
			continue
		}
		if lang.IsCommentLine(line) && !lang.IsPackageDeclaration(line) {
			findings = append(findings, a.reporter.At(
				fc.Path,
				lineNum,
				"file starts with a comment before the package declaration",
				"remove the comment; no package docstrings",
			))
		}
	}
	return findings
}

func isExemptHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "// Code generated") ||
		strings.HasPrefix(trimmed, "//go:build") ||
		strings.HasPrefix(trimmed, "// +build")
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
