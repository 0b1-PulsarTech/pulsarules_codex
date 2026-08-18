package naming

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports identifiers that violate project naming conventions:
// sequential numbered names, Hungarian notation, and noise words.
type Analyzer struct{}

var namingReporter = core.NewReporter("naming", core.SeverityWarning, core.CatSyntax)

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "naming" }
func (a *Analyzer) Name() string { return "Naming conventions" }
func (a *Analyzer) Description() string {
	return "Reports numbered names, Hungarian notation, noise words, and misleading names"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageStatic }
func (a *Analyzer) Category() core.Category { return core.CatSyntax }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	f *ast.File,
) []core.Finding {
	var findings []core.Finding
	// Two passes: the sequential-name and noise rules both need the whole
	// file (siblings, and the types names are built from), which no single
	// identifier can answer for itself.
	checker := fileChecker{
		index: newFileIndex(f),
		emit: func(pos token.Pos, msg, suggestion string) {
			findings = append(
				findings,
				namingReporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
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
