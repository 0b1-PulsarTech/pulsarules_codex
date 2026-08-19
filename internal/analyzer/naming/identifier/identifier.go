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

func (a *Analyzer) ID() string          { return "naming" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc pass.FileChange,
	f *ast.File,
) []pass.Finding {
	var findings []pass.Finding
	// Two passes: the sequential-name and noise rules both need the whole
	// file (siblings, and the types names are built from), which no single
	// identifier can answer for itself.
	checker := fileChecker{
		index: newFileIndex(f),
		emit: func(pos token.Pos, msg, suggestion string) {
			findings = append(
				findings,
				a.reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
			)
		},
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			checker.funcDecl(v)
		case *ast.GenDecl:
			checker.genDecl(v)
		case *ast.AssignStmt:
			if v.Tok == token.DEFINE {
				for _, lhs := range v.Lhs {
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
