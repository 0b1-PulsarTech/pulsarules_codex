package importgroup

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: pass.NewReporter("import-groups", pass.SeverityError, pass.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "import-groups" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }
func (a *Analyzer) Needs() pass.Requirements {
	return pass.Requirements{}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	modulePath, err := pass.ModulePath(ctx.ProjectDir)
	if err != nil || modulePath == "" {
		return nil
	}
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), modulePath, fc, f)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	modulePath string,
	fc pass.FileChange,
	f *ast.File,
) []pass.Finding {
	if len(f.Imports) == 0 {
		return nil
	}

	type group int
	const (
		groupStd group = iota
		groupExt
		groupModule
	)

	classify := func(path string) group {
		clean := strings.Trim(path, `"`)
		if modulePath != "" &&
			(clean == modulePath || strings.HasPrefix(clean, modulePath+"/")) {
			return groupModule
		}
		if strings.Contains(clean, ".") {
			return groupExt
		}
		return groupStd
	}

	prevGroup := classify(f.Imports[0].Path.Value)

	for _, spec := range f.Imports[1:] {
		g := classify(spec.Path.Value)
		if g < prevGroup {
			return []pass.Finding{
				a.reporter.At(
					fc.Path,
					fset.Position(spec.Pos()).Line,
					fmt.Sprintf(
						"import %s out of order (stdlib → external → %s)",
						spec.Path.Value,
						modulePath,
					),
					"reorder imports to follow: stdlib, external, this module",
				),
			}
		}
		prevGroup = g
	}
	return nil
}
