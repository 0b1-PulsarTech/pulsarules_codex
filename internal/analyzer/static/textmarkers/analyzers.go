package textmarkers

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

type textReporters struct {
	carrier     core.Reporter
	contextual  core.Reporter
	frontmatter core.Reporter
}

type TextAnalyzer struct {
	reporters textReporters
}

func NewTextAnalyzer() *TextAnalyzer {
	return &TextAnalyzer{reporters: textReporters{
		// Classes `clean --write` removes outright: no neighbouring context can justify
		// them, and the bidi controls among them are the Trojan Source vector
		// (CVE-2021-42574), so they block.
		carrier: core.NewReporter("text-markers", core.SeverityError, core.CatSyntax),
		// Invisible characters that may be load-bearing (emoji glue, script joiners)
		// cannot be judged here, so they advise instead of blocking.
		contextual:  core.NewReporter("text-markers", core.SeverityWarning, core.CatSyntax),
		frontmatter: core.NewReporter("ai-frontmatter", core.SeverityError, core.CatSyntax),
	}}
}

func (a *TextAnalyzer) ID() string { return "text-markers" }

func (a *TextAnalyzer) Stage() core.StageID { return core.StageStatic }

func (a *TextAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return eachMarkedFile(ctx, func(fc core.FileChange, src string) []core.Finding {
		return checkCarriers(fc, src, a.reporters)
	})
}

func checkCarriers(fc core.FileChange, src string, reporters textReporters) []core.Finding {
	findings := frontmatterFindings(fc, src, reporters.frontmatter)
	for _, found := range mark.Scan(src) {
		if found.Class == mark.ClassTypographic {
			continue
		}
		// The class picks the reporter: a carrier no context can justify blocks, while
		// one that may be load-bearing only advises.
		reporter := reporters.carrier
		if found.Class == mark.ClassContextual {
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

func carrierAdvice(class mark.Class) string {
	if class == mark.ClassContextual {
		return "it may be load-bearing here; judge it, then remove it by hand"
	}
	return "run `pulsarules_cli clean --write`, which removes this class"
}

type TypographicAnalyzer struct{}

func NewTypographicAnalyzer() *TypographicAnalyzer { return &TypographicAnalyzer{} }

func (a *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (a *TypographicAnalyzer) Stage() core.StageID { return core.StageStatic }

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
