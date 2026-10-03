package topoffile

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	langs *core.LanguageRegistry
}

func NewAnalyzer(langs *core.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs}
}

func (a *Analyzer) ID() string          { return "top-of-file" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.langs == nil {
		return nil
	}
	// Required header lines (core.ExemptHeaderSpan) are skipped so a hand comment after
	// them is still flagged; a core.IsContentStartLine line ends the scan, and a
	// core.IsGeneratedMarkerLine line before it exempts the file in full.
	extFilter := core.ExtensionFilter(ctx, a.ID(), defaultExtensions())
	eligible := func(fc core.FileChange) bool {
		return extFilter(fc) && a.langs.Lookup(fc.Extension) != nil
	}
	return core.EachChangedFile(ctx, eligible, func(
		fc core.FileChange, src []byte,
	) []core.Finding {
		lineNum, ok := leadingCommentLine(a.langs.Lookup(fc.Extension), fc.Extension, src)
		if !ok {
			return nil
		}
		return []core.Finding{reporterFor(ctx, a.ID(), fc.Extension).At(
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

func reporterFor(ctx *core.AnalysisContext, id, extension string) core.Reporter {
	severity := core.SeverityWarning
	if strings.EqualFold(extension, ".go") {
		severity = core.SeverityError
	}
	return core.NewReporter(id, severity, core.CatSyntax).Resolved(ctx)
}
