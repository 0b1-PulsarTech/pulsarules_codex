package singleletter

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "single-letter-name" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) reporter() pass.Reporter {
	return pass.NewReporter(analyzer.ID(), pass.SeverityWarning, pass.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	cfg := settings{
		reporter:     pass.ResolveReporter(analyzer.reporter(), ctx),
		exportedOnly: ctx.Params(analyzer.ID()).Bool("exported_only", false),
	}
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, cfg)
	})
}

type settings struct {
	reporter     pass.Reporter
	exportedOnly bool
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, cfg settings,
) []pass.Finding {
	scan := newCollector(testingNames(file))
	scan.exportedOnly = cfg.exportedOnly
	scan.add(file.Name, kindPackage)
	ast.Inspect(file, scan.visit)
	findings := make([]pass.Finding, 0, len(scan.sites))
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
