package character

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

type textReporters struct {
	carrier     finding.Reporter
	contextual  finding.Reporter
	frontmatter finding.Reporter
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
		contextual: finding.NewReporter(
			"text-markers",
			finding.SeverityWarning,
			finding.CatSyntax,
		),
		frontmatter: finding.NewReporter(
			"ai-frontmatter",
			finding.SeverityError,
			finding.CatSyntax,
		),
	}}
}

func (analyzer *TextAnalyzer) ID() string { return "text-markers" }

func (analyzer *TextAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *TextAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporters := textReporters{
		carrier:     pass.ResolveReporter(analyzer.reporters.carrier, ctx),
		contextual:  pass.ResolveReporter(analyzer.reporters.contextual, ctx),
		frontmatter: pass.ResolveReporter(analyzer.reporters.frontmatter, ctx),
	}
	eligible := pass.ExtensionFilter(ctx, analyzer.ID(), markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc pass.FileChange, src string) []finding.Finding {
		return checkCarriers(fc, src, reporters)
	})
}

func checkCarriers(fc pass.FileChange, src string, reporters textReporters) []finding.Finding {
	findings := frontmatterFindings(fc, src, reporters.frontmatter)
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

type TypographicAnalyzer struct {
	reporter finding.Reporter
}

func NewTypographicAnalyzer() *TypographicAnalyzer {
	return &TypographicAnalyzer{
		// Blocks by default; a project that treats ASCII punctuation as house style
		// lowers it through the analyzer's "severity" param.
		reporter: finding.NewReporter(
			"typographic-markers",
			finding.SeverityError,
			finding.CatSyntax,
		),
	}
}

func (analyzer *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (analyzer *TypographicAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *TypographicAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	eligible := pass.ExtensionFilter(ctx, analyzer.ID(), markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc pass.FileChange, src string) []finding.Finding {
		return checkTypographic(fc, src, reporter)
	})
}

func checkTypographic(fc pass.FileChange, src string, reporter finding.Reporter) []finding.Finding {
	var code []region
	if strings.EqualFold(fc.Extension, ".md") {
		code = codeRegions(src)
	}

	var findings []finding.Finding
	for _, found := range invisible.Scan(src) {
		if found.Class != invisible.ClassTypographic || inCode(code, found.Offset) {
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
