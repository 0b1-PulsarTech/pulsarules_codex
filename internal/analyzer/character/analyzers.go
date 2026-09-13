package character

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

type textReporters struct {
	carrier     pass.Reporter
	contextual  pass.Reporter
	frontmatter pass.Reporter
}

type TextAnalyzer struct {
	reporters textReporters
}

func NewTextAnalyzer() *TextAnalyzer {
	return &TextAnalyzer{reporters: textReporters{
		// Classes `clean --write` removes outright: no neighbouring context can justify
		// them, and the bidi controls among them are the Trojan Source vector
		// (CVE-2021-42574), so they block.
		carrier: pass.NewReporter("text-markers", pass.SeverityError, pass.CatSyntax),
		// Invisible characters that may be load-bearing (emoji glue, script joiners)
		// cannot be judged here, so they advise instead of blocking.
		contextual:  pass.NewReporter("text-markers", pass.SeverityWarning, pass.CatSyntax),
		frontmatter: pass.NewReporter("ai-frontmatter", pass.SeverityError, pass.CatSyntax),
	}}
}

func (a *TextAnalyzer) ID() string { return "text-markers" }

func (a *TextAnalyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *TextAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporters := textReporters{
		carrier:     pass.ResolveReporter(a.reporters.carrier, ctx),
		contextual:  pass.ResolveReporter(a.reporters.contextual, ctx),
		frontmatter: pass.ResolveReporter(a.reporters.frontmatter, ctx),
	}
	return eachMarkedFile(ctx, a.ID(), func(fc pass.FileChange, src string) []pass.Finding {
		return checkCarriers(fc, src, reporters)
	})
}

func checkCarriers(fc pass.FileChange, src string, reporters textReporters) []pass.Finding {
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
	reporter pass.Reporter
}

func NewTypographicAnalyzer() *TypographicAnalyzer {
	return &TypographicAnalyzer{
		// Blocks by default; a project that treats ASCII punctuation as house style
		// lowers it through the analyzer's "severity" param.
		reporter: pass.NewReporter("typographic-markers", pass.SeverityError, pass.CatSyntax),
	}
}

func (a *TypographicAnalyzer) ID() string { return "typographic-markers" }

func (a *TypographicAnalyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *TypographicAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(a.reporter, ctx)
	return eachMarkedFile(ctx, a.ID(), func(fc pass.FileChange, src string) []pass.Finding {
		return checkTypographic(fc, src, reporter)
	})
}

func checkTypographic(fc pass.FileChange, src string, reporter pass.Reporter) []pass.Finding {
	var code []region
	if strings.EqualFold(fc.Extension, ".md") {
		code = codeRegions(src)
	}

	var findings []pass.Finding
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
