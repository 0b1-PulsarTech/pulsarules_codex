package character

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

const (
	TextID        = "text-markers"
	TypographicID = "typographic-markers"
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
		carrier: finding.NewReporter(TextID, finding.SeverityError, finding.CatSyntax),
		// Invisible characters that may be load-bearing (emoji glue, script joiners)
		// cannot be judged here, so they advise instead of blocking.
		contextual: finding.NewReporter(TextID, finding.SeverityWarning, finding.CatSyntax),
		frontmatter: finding.NewReporter(
			"ai-frontmatter",
			finding.SeverityError,
			finding.CatSyntax,
		),
	}}
}

func (analyzer *TextAnalyzer) ID() string { return TextID }

func (analyzer *TextAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *TextAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporters := textReporters{
		carrier:     pass.ResolveReporter(analyzer.reporters.carrier, ctx),
		contextual:  pass.ResolveReporter(analyzer.reporters.contextual, ctx),
		frontmatter: pass.ResolveReporter(analyzer.reporters.frontmatter, ctx),
	}
	eligible := pass.ExtensionFilter(ctx, TextID, markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc pass.FileChange, src string) []finding.Finding {
		return checkCarriers(fc, src, reporters)
	})
}

func checkCarriers(fc pass.FileChange, src string, reporters textReporters) []finding.Finding {
	findings := frontmatterFindings(fc, src, reporters.frontmatter)
	// The class picks the reporter: only a carrier no context can justify blocks.
	notTypographic := func(class invisible.Class) bool { return class != invisible.ClassTypographic }
	hinted := hintMarks(fc, src, notTypographic, func(
		found invisible.Mark, literals language.Literals, code markdown.Code,
	) (finding.Reporter, hint, bool) {
		reporter := reporters.carrier
		if found.Class == invisible.ClassContextual {
			reporter = reporters.contextual
		}
		return reporter, carrierAdvice(found, fc.Extension, literals, code), false
	})
	return append(findings, settle(hinted)...)
}

type TypographicAnalyzer struct {
	reporter finding.Reporter
}

func NewTypographicAnalyzer() *TypographicAnalyzer {
	return &TypographicAnalyzer{
		// Blocks by default; a project that treats ASCII punctuation as house style
		// lowers it with the analyzer's "severity" key.
		reporter: finding.NewReporter(TypographicID, finding.SeverityError, finding.CatSyntax),
	}
}

func (analyzer *TypographicAnalyzer) ID() string { return TypographicID }

func (analyzer *TypographicAnalyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *TypographicAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	eligible := pass.ExtensionFilter(ctx, TypographicID, markedExtensions())
	return eachMarkedFile(ctx, eligible, func(fc pass.FileChange, src string) []finding.Finding {
		return checkTypographic(fc, src, reporter)
	})
}

func checkTypographic(fc pass.FileChange, src string, reporter finding.Reporter) []finding.Finding {
	typographic := func(class invisible.Class) bool { return class == invisible.ClassTypographic }
	return settle(hintMarks(fc, src, typographic, func(
		found invisible.Mark, literals language.Literals, code markdown.Code,
	) (finding.Reporter, hint, bool) {
		return reporter, typographicAdvice(found, fc.Extension, literals, code),
			code.Certain(found.Offset)
	}))
}

func hintMarks(
	fc pass.FileChange, src string, wanted func(invisible.Class) bool,
	advise func(invisible.Mark, language.Literals, markdown.Code) (finding.Reporter, hint, bool),
) []hintedFinding {
	var marks []invisible.Mark
	// Lexing a file is the costly step, so a file with no mark of this kind is never lexed.
	for _, found := range invisible.Scan(src) {
		if wanted(found.Class) {
			marks = append(marks, found)
		}
	}
	if len(marks) == 0 {
		return nil
	}
	literals := language.ScanLiterals(fc.Extension, src)
	code := markdown.MarkdownCode(fc.Extension, src)
	var hinted []hintedFinding
	for _, found := range marks {
		reporter, advice, skip := advise(found, literals, code)
		if skip {
			continue
		}
		hinted = append(hinted, hintedFinding{
			finding: reporter.At(
				fc.Path, found.Line, fmt.Sprintf("%s (U+%04X)", found.Name, found.Rune),
				advice.text,
			),
			caution: advice.caution,
		})
	}
	return hinted
}

func DescribeText() pass.Descriptor {
	return pass.Descriptor{
		ID:     TextID,
		Rule:   "source-files",
		Params: []string{pass.ParamExtensions},
	}
}

func DescribeTypographic() pass.Descriptor {
	return pass.Descriptor{
		ID:     TypographicID,
		Rule:   "source-files",
		Params: []string{pass.ParamExtensions},
	}
}
