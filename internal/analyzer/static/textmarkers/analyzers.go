package textmarkers

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

var (
	// carrierReporter covers the classes `clean --write` removes outright: no
	// neighbouring context can justify them, and the bidi controls among them
	// are the Trojan Source vector (CVE-2021-42574), so they block.
	carrierReporter = core.NewReporter("text-markers", core.SeverityError, core.CatSyntax)
	// contextualReporter covers the invisible characters that MAY be
	// load-bearing - emoji glue, script joiners - which this analyzer is not
	// able to judge, so they advise instead of blocking.
	contextualReporter = core.NewReporter("text-markers", core.SeverityWarning, core.CatSyntax)
)

// TextAnalyzer reports invisible carriers and exotic spaces.
type TextAnalyzer struct{}

func NewTextAnalyzer() *TextAnalyzer { return &TextAnalyzer{} }

func (a *TextAnalyzer) ID() string   { return "text-markers" }
func (a *TextAnalyzer) Name() string { return "Text markers" }

func (a *TextAnalyzer) Description() string {
	return "Reports invisible characters, exotic spaces, and AI provenance frontmatter"
}
func (a *TextAnalyzer) Stage() core.StageID     { return core.StageStatic }
func (a *TextAnalyzer) Category() core.Category { return core.CatSyntax }

func (a *TextAnalyzer) Needs() core.Requirements { return core.Requirements{} }

func (a *TextAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return eachMarkedFile(ctx, checkCarriers)
}

func checkCarriers(fc core.FileChange, src string) []core.Finding {
	findings := frontmatterFindings(fc, src)
	for _, found := range mark.Scan(src) {
		if found.Class == mark.ClassTypographic {
			continue
		}
		reporter := carrierReporter
		if found.Class == mark.ClassContextual {
			reporter = contextualReporter
		}
		findings = append(findings, reporter.At(
			fc.Path, found.Line,
			fmt.Sprintf("%s (U+%04X)", found.Name, found.Rune),
			carrierAdvice(found.Class),
		))
	}
	return findings
}

func carrierAdvice(class mark.Class) string {
	if class == mark.ClassContextual {
		return "it may be load-bearing here; judge it, then remove it by hand"
	}
	return "run `pulsarules_cli clean --write`, which removes this class"
}

// TypographicAnalyzer reports the punctuation an AI model reaches for by default.
type TypographicAnalyzer struct{}

func NewTypographicAnalyzer() *TypographicAnalyzer { return &TypographicAnalyzer{} }

func (a *TypographicAnalyzer) ID() string   { return "typographic-markers" }
func (a *TypographicAnalyzer) Name() string { return "Typographic markers" }

func (a *TypographicAnalyzer) Description() string {
	return "Reports em/en dashes, ellipses, and curly quotes; use the ASCII form"
}
func (a *TypographicAnalyzer) Stage() core.StageID     { return core.StageStatic }
func (a *TypographicAnalyzer) Category() core.Category { return core.CatSyntax }

func (a *TypographicAnalyzer) Needs() core.Requirements { return core.Requirements{} }

// Analyze builds its reporter per run: the default blocks, and a project that
// treats ASCII punctuation as house style rather than a defect sets the
// analyzer's "severity" param to "warning" to keep the report without the gate.
func (a *TypographicAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := core.NewReporter(
		a.ID(),
		ctx.Params(a.ID()).Severity(core.SeverityError),
		core.CatSyntax,
	)
	return eachMarkedFile(ctx, func(fc core.FileChange, src string) []core.Finding {
		return checkTypographic(reporter, fc, src)
	})
}

// why: never auto-fixed, only reported. Inside a string literal or a fenced
// block the character is data, and no analyzer can tell that from prose.
func checkTypographic(reporter core.Reporter, fc core.FileChange, src string) []core.Finding {
	var findings []core.Finding
	for _, found := range mark.Scan(src) {
		if found.Class != mark.ClassTypographic {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path, found.Line,
			fmt.Sprintf("%s (U+%04X)", found.Name, found.Rune),
			"replace it with the ASCII form by hand",
		))
	}
	return findings
}
