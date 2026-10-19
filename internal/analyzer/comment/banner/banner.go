package banner

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	langs *pass.LanguageRegistry
}

func NewAnalyzer(langs *pass.LanguageRegistry) *Analyzer {
	return &Analyzer{langs: langs}
}

func (analyzer *Analyzer) ID() string             { return "top-of-file" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if analyzer.langs == nil {
		return nil
	}
	// Required header lines (lexer.ExemptHeaderSpan) are skipped so a hand comment after
	// them is still flagged; a lexer.IsContentStartLine line ends the scan, and a
	// generated file is exempt in full.
	extFilter := pass.ExtensionFilter(ctx, analyzer.ID(), defaultExtensions())
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && !fc.Generated && analyzer.langs.Lookup(fc.Extension) != nil
	}
	scan := func(fc pass.FileChange, src []byte) []finding.Finding {
		lang := analyzer.langs.Lookup(fc.Extension)
		lineNum, ok := leadingCommentLine(lang, fc.Extension, src)
		if !ok {
			return nil
		}
		return []finding.Finding{reporterFor(ctx, analyzer.ID(), fc.Extension).At(
			fc.Path,
			lineNum,
			"file starts with a comment",
			"remove the leading comment; no file or package docstrings",
		)}
	}
	return pass.EachChangedFile(ctx, eligible, scan)
}

func defaultExtensions() []string {
	return []string{
		".go", ".proto", ".sql", ".nix", ".svelte", ".ts", ".dart",
		".js", ".mjs", ".cjs", ".jsx", ".tsx",
	}
}

func reporterFor(ctx *pass.Context, id, extension string) finding.Reporter {
	severity := finding.SeverityWarning
	if strings.EqualFold(extension, ".go") {
		severity = finding.SeverityError
	}
	return pass.ResolveReporter(finding.NewReporter(id, severity, finding.CatSyntax), ctx)
}
