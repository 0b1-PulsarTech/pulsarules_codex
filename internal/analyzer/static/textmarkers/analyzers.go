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
	// typoReporter blocks by default; a project that treats ASCII punctuation
	// as house style rather than a defect lowers it through the severity param.
	typoReporter = core.NewReporter("typographic-markers", core.SeverityError, core.CatSyntax)
)

// TextAnalyzer reports invisible carriers and exotic spaces.
type TextAnalyzer struct{}

func NewTextAnalyzer() *TextAnalyzer { return &TextAnalyzer{} }

func (a *TextAnalyzer) ID() string { return "text-markers" }

func (a *TextAnalyzer) Stage() core.StageID { return core.StageStatic }

func (a *TextAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	carrier := carrierReporter.Resolved(ctx)
	contextual := contextualReporter.Resolved(ctx)
	frontReporter := frontmatterReporter.Resolved(ctx)
	return eachMarkedFile(ctx, func(fc core.FileChange, src string) []core.Finding {
		return checkCarriers(fc, src, carrier, contextual, frontReporter)
	})
}

// why: the class picks the reporter - a carrier no context can justify blocks,
// while one that MAY be load-bearing only advises (see the var block).
func checkCarriers(
	fc core.FileChange, src string, carrier, contextual, frontReporter core.Reporter,
) []core.Finding {
	findings := frontmatterFindings(fc, src, frontReporter)
	for _, found := range mark.Scan(src) {
		if found.Class == mark.ClassTypographic {
			continue
		}
		reporter := carrier
		if found.Class == mark.ClassContextual {
			reporter = contextual
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

func (a *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (a *TypographicAnalyzer) Stage() core.StageID { return core.StageStatic }

// Analyze builds its reporter per run: the default blocks, and a project that
// treats ASCII punctuation as house style rather than a defect sets the
// analyzer's "severity" param to "warning" to keep the report without the gate.
func (a *TypographicAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := typoReporter.Resolved(ctx)
	return eachMarkedFile(ctx, func(fc core.FileChange, src string) []core.Finding {
		return checkTypographic(fc, src, reporter)
	})
}

// why: never auto-fixed, only reported. Inside a string literal or a fenced
// block the character is data, and no analyzer can tell that from prose.
func checkTypographic(fc core.FileChange, src string, reporter core.Reporter) []core.Finding {
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
