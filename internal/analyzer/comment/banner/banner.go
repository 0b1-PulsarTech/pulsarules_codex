package banner

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const ID = "top-of-file"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	extFilter := pass.ExtensionFilter(ctx, ID, defaultExtensions())
	// Required header lines (lexer.ExemptHeaderSpan) are skipped so a hand comment after
	// them is still flagged; a lexer.IsContentStartLine line ends the scan, and a
	// generated file is exempt in full.
	eligible := func(fc pass.FileChange) bool {
		return extFilter(fc) && !fc.Generated && language.ForExtension(fc.Extension).ID != ""
	}
	scan := func(fc pass.FileChange, src []byte) []finding.Finding {
		lineNum, ok := leadingCommentLine(fc.Extension, src)
		if !ok {
			return nil
		}
		return []finding.Finding{reporterFor(fc.Extension).At(
			fc.Path,
			lineNum,
			"file starts with a comment",
			"remove the leading comment; no file or package docstrings",
		)}
	}
	return ctx.EachSource(eligible, scan)
}

func defaultExtensions() []string {
	return language.Extensions(
		language.Go, language.Proto, language.SQL, language.Nix, language.Svelte,
		language.TypeScript, language.Dart, language.JavaScript,
	)
}

func reporterFor(extension string) finding.Reporter {
	severity := finding.SeverityWarning
	if language.ForExtension(extension).ID == language.Go {
		severity = finding.SeverityError
	}
	return finding.NewReporter(ID, severity, finding.CatSyntax)
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "comments",
		Params: []string{pass.ParamExtensions},
	}
}
