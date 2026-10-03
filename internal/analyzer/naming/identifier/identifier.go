package identifier

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: pass.NewReporter("naming", pass.SeverityWarning, pass.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string          { return "naming" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		return analyzer.checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func (analyzer *Analyzer) checkFile(
	fset *token.FileSet,
	fc pass.FileChange,
	file *ast.File,
	reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	// Two passes: the sequential-name and noise rules both need the whole
	// file (siblings, and the types names are built from), which no single
	// identifier can answer for itself.
	checker := fileChecker{
		index: newFileIndex(file),
		emit: func(pos token.Pos, msg, suggestion string) {
			findings = append(
				findings,
				reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
			)
		},
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			checker.funcDecl(typed)
		case *ast.GenDecl:
			checker.genDecl(typed)
		case *ast.AssignStmt:
			if typed.Tok == token.DEFINE {
				for _, lhs := range typed.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
						checker.check(id.Name, id.NamePos)
					}
				}
			}
		}
		return true
	})

	return findings
}
