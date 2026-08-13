package imports

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter("import-groups", core.SeverityError, core.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "import-groups" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	modulePath, err := core.ModulePath(ctx.ProjectDir)
	if err != nil || modulePath == "" {
		return nil
	}
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), modulePath, fc, f)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	modulePath string,
	fc core.FileChange,
	f *ast.File,
) []core.Finding {
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
			return []core.Finding{
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
