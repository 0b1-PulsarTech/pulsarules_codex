package doccomment

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string { return "doc-comment" }

func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter().Resolved(ctx)
	return syntaxfiles.Each(ctx, func(fc core.FileChange, file *syntax.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc core.FileChange, file *syntax.File, reporter core.Reporter) []core.Finding {
	var findings []core.Finding
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
	case syntax.DocFunction:
		return "delete it; keep a non-obvious why on the line it explains inside the body, " +
			"never as a multi-line block at the top of the body"
	case syntax.DocMethod:
		return "delete it; the method name and signature carry the meaning"
	case syntax.DocType:
		return "delete it; the type name and fields carry the meaning"
	}
	return "delete it; the name carries the meaning"
}
