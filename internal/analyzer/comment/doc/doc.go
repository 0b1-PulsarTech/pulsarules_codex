package doc

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const ID = "doc-comment"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc pass.FileChange, file *syntax.File, reporter finding.Reporter) []finding.Finding {
	var findings []finding.Finding
	for _, site := range file.DocSites {
		if site.Exempt {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path,
			site.Line,
			fmt.Sprintf("%s carries a doc comment", site.Symbol),
			hintFor(site),
		))
	}
	return findings
}

func hintFor(site syntax.DocSite) string {
	if site.MixedDirective {
		return "delete the prose lines, keep the directive lines"
	}
	switch site.Kind {
	case syntax.DocFunction, syntax.DocConstructor:
		return "delete it; keep a non-obvious why on the line it explains inside the body, " +
			"never as a multi-line block at the top of the body"
	case syntax.DocMethod:
		return "delete it; the method name and signature carry the meaning"
	case syntax.DocType:
		return "delete it; the type name and fields carry the meaning"
	}
	return "delete it; the name carries the meaning"
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "effective-go"}
}
