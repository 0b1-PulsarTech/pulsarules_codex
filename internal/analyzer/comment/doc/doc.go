package doc

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string { return "doc-comment" }

func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) reporter() pass.Reporter {
	return pass.NewReporter(analyzer.ID(), pass.SeverityWarning, pass.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []pass.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc pass.FileChange, file *syntax.File, reporter pass.Reporter) []pass.Finding {
	var findings []pass.Finding
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
