package topoffile

import (
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
	if a.langs == nil {
		return nil
	}
	reporter := a.reporter.Resolved(ctx)
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
		return []core.Finding{reporter.At(
			fc.Path,
			lineNum,
			"file starts with a comment before the package declaration",
			"remove the comment; no package docstrings",
		)}
	})
}

func defaultExtensions() []string {
	return []string{".go", ".proto", ".sql", ".nix", ".svelte", ".ts", ".dart"}
}
