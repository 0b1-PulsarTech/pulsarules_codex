package topoffile

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports files where the first non-blank line is a comment
// rather than the package declaration.
type Analyzer struct {
	langs *core.LanguageRegistry
}

var topOfFileReporter = core.NewReporter("top-of-file", core.SeverityError, core.CatSyntax)

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil {
		return nil
	}
	reporter := topOfFileReporter.Resolved(ctx)
	eligible := func(fc core.FileChange) bool { return a.langs.Lookup(fc.Extension) != nil }
	return core.EachChangedFile(ctx, eligible, func(fc core.FileChange, src []byte) []core.Finding {
		lang := a.langs.Lookup(fc.Extension)
		line, lineNum, ok := firstNonBlank(src)
		if !ok || core.IsExemptHeaderLine(line) {
			return nil
		}
		if lang.IsCommentLine(line) && !lang.IsPackageDeclaration(line) {
			return []core.Finding{reporter.At(
				fc.Path,
				lineNum,
				"file starts with a comment before the package declaration",
				"remove the comment; no package docstrings",
			)}
		}
		return nil
	})
}

// firstNonBlank returns the first non-blank line from src, its 1-based line
// number, and whether a non-blank line exists.
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
