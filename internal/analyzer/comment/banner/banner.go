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
	if a.langs == nil {
		return nil
	}
	reporter := pass.ResolveReporter(a.reporter, ctx)
	eligible := func(fc pass.FileChange) bool { return a.langs.Lookup(fc.Extension) != nil }
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []pass.Finding {
		lang := a.langs.Lookup(fc.Extension)
		line, lineNum, ok := firstNonBlank(src)
		if !ok || pass.IsExemptHeaderLine(line) {
			return nil
		}
		scanner := pass.NewCommentScanner(lang.CommentSyntax())
		if scanner.IsCommentLine(line) && !lang.IsPackageDeclaration(line) {
			return []pass.Finding{reporter.At(
				fc.Path,
				lineNum,
				"file starts with a comment before the package declaration",
				"remove the comment; no package docstrings",
			)}
		}
		return nil
	})
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
