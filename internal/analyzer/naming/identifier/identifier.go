package identifier

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "naming"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		return analyzer.checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func (analyzer *Analyzer) checkFile(
	fset *token.FileSet,
	fc pass.FileChange,
	file *ast.File,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	// The sequential and noise rules need the whole file before any name is judged.
	checker := fileChecker{
		index: newFileIndex(file),
		emit: func(pos token.Pos, msg, suggestion string) {
			findings = append(
				findings,
				reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
			)
		},
	}

	checker.run()

	return findings
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "naming"}
}
