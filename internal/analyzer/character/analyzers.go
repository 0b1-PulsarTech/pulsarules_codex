package character

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

type textReporters struct {
	carrier    finding.Reporter
	contextual finding.Reporter
}

type TextAnalyzer struct {
	reporters textReporters
}

func NewTextAnalyzer() *TextAnalyzer {
	return &TextAnalyzer{reporters: textReporters{
		// Classes `clean --write` removes outright: no neighbouring context can justify
		// them, and the bidi controls among them are the Trojan Source vector
		// (CVE-2021-42574), so they block.
		carrier: finding.NewReporter("text-markers", finding.SeverityError, finding.CatSyntax),
		// Invisible characters that may be load-bearing (emoji glue, script joiners)
		// cannot be judged here, so they advise instead of blocking.
		contextual: finding.NewReporter("text-markers", finding.SeverityWarning, finding.CatSyntax),
	}}
}

func (a *TextAnalyzer) ID() string { return "text-markers" }

func (a *TextAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (a *TextAnalyzer) Needs() finding.Requirements { return finding.Requirements{} }

func (a *TextAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	return eachMarkedFile(ctx, func(fc pass.FileChange, src string) []finding.Finding {
		return checkCarriers(fc, src, a.reporters)
	})
}

func checkCarriers(fc pass.FileChange, src string, reporters textReporters) []finding.Finding {
	var findings []finding.Finding
	for _, found := range invisible.Scan(src) {
		if found.Class == invisible.ClassTypographic {
			continue
		}
		// The class picks the reporter: a carrier no context can justify blocks, while
		// one that may be load-bearing only advises.
		reporter := reporters.carrier
		if found.Class == invisible.ClassContextual {
			reporter = reporters.contextual
		}
		findings = append(findings, reporter.At(
			fc.Path, found.Line,
			fmt.Sprintf("%s (U+%04X)", found.Name, found.Rune),
			carrierAdvice(found.Class),
		))
	}
	return findings
}

func carrierAdvice(class invisible.Class) string {
	if class == invisible.ClassContextual {
		return "it may be load-bearing here; judge it, then remove it by hand"
	}
	return "run `pulsarules_cli clean --write`, which removes this class"
}

type TypographicAnalyzer struct{}

func NewTypographicAnalyzer() *TypographicAnalyzer { return &TypographicAnalyzer{} }

func (a *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (a *TypographicAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (a *TypographicAnalyzer) Needs() finding.Requirements { return finding.Requirements{} }

func (a *TypographicAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := finding.NewReporter(
		a.ID(),
		ctx.Params(a.ID()).Severity(finding.SeverityError),
		finding.CatSyntax,
	)
	return eachMarkedFile(ctx, func(fc pass.FileChange, src string) []finding.Finding {
		return checkTypographic(reporter, fc, src)
	})
}

func checkTypographic(reporter finding.Reporter, fc pass.FileChange, src string) []finding.Finding {
	var findings []finding.Finding
	for _, found := range invisible.Scan(src) {
		if found.Class != invisible.ClassTypographic {
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
