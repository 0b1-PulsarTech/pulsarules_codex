package banner

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	langs *pass.LanguageRegistry
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if a.langs == nil {
		return nil
	}
	// Required header lines (core.ExemptHeaderSpan) are skipped so a hand comment after
	// them is still flagged; a core.IsContentStartLine line ends the scan, and a
	// core.IsGeneratedMarkerLine line before it exempts the file in full.
	extFilter := pass.ExtensionFilter(ctx, a.ID(), defaultExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []pass.Finding {
		lineNum, ok := leadingCommentLine(a.langs.Lookup(fc.Extension), fc.Extension, src)
		if !ok {
			return nil
		}
		return []pass.Finding{reporterFor(ctx, a.ID(), fc.Extension).At(
			fc.Path,
			lineNum,
			"file starts with a comment",
			"remove the leading comment; no file or package docstrings",
		)}
	})
}

func defaultExtensions() []string {
	return []string{".go", ".proto", ".sql", ".nix", ".svelte", ".ts", ".dart"}
}

func reporterFor(ctx *pass.Context, id, extension string) pass.Reporter {
	severity := pass.SeverityWarning
	if strings.EqualFold(extension, ".go") {
		severity = pass.SeverityError
	}
	return pass.ResolveReporter(pass.NewReporter(id, severity, pass.CatSyntax), ctx)
}
