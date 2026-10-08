package textmarkers

import (
	"fmt"
	"strings"

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

func (analyzer *TextAnalyzer) ID() string { return "text-markers" }

func (analyzer *TextAnalyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *TextAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporters := textReporters{
		carrier:     analyzer.reporters.carrier.Resolved(ctx),
		contextual:  analyzer.reporters.contextual.Resolved(ctx),
		frontmatter: analyzer.reporters.frontmatter.Resolved(ctx),
	}
	eligible := core.ExtensionFilter(ctx, analyzer.ID(), markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc core.FileChange, src string) []core.Finding {
		return checkCarriers(fc, src, reporters)
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

type TypographicAnalyzer struct {
	reporter core.Reporter
}

func NewTypographicAnalyzer() *TypographicAnalyzer {
	return &TypographicAnalyzer{
		// Blocks by default; a project that treats ASCII punctuation as house style
		// lowers it through the analyzer's "severity" param.
		reporter: core.NewReporter("typographic-markers", core.SeverityError, core.CatSyntax),
	}
}

func (analyzer *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (analyzer *TypographicAnalyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *TypographicAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter.Resolved(ctx)
	eligible := core.ExtensionFilter(ctx, analyzer.ID(), markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc core.FileChange, src string) []core.Finding {
		return checkTypographic(fc, src, reporter)
	})
}

func checkTypographic(fc core.FileChange, src string, reporter core.Reporter) []core.Finding {
	var code []region
	if strings.EqualFold(fc.Extension, ".md") {
		code = codeRegions(src)
	}

	var findings []core.Finding
	for _, found := range mark.Scan(src) {
		if found.Class != mark.ClassTypographic || inCode(code, found.Offset) {
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
