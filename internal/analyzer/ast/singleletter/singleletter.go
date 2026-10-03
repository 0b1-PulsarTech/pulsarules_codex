package singleletter

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "single-letter-name" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	cfg := settings{
		reporter:     analyzer.reporter().Resolved(ctx),
		exportedOnly: ctx.Params(analyzer.ID()).Bool("exported_only", false),
	}
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, cfg)
	})
}

type settings struct {
	reporter     core.Reporter
	exportedOnly bool
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, file *ast.File, cfg settings,
) []core.Finding {
	scan := newCollector(testingNames(file))
	scan.exportedOnly = cfg.exportedOnly
	scan.add(file.Name, kindPackage)
	ast.Inspect(file, scan.visit)
	findings := make([]core.Finding, 0, len(scan.sites))
	for _, found := range scan.sites {
		findings = append(findings, cfg.reporter.At(
			fc.Path,
			fset.Position(found.ident.Pos()).Line,
			fmt.Sprintf("%s %q is a single letter", found.kind, found.ident.Name),
			"name it after what it is or holds; a short abbreviation such as srv, row "+
				"or idx is fine, one letter is not",
		))
	}
	return findings
}
