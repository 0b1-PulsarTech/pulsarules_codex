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

func (analyzer *Analyzer) ID() string          { return "import-groups" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	modulePath, err := core.ModulePath(ctx.ProjectDir)
	if err != nil || modulePath == "" {
		return nil
	}
	reporter := analyzer.reporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		return analyzer.checkFile(ctx.ASTCache.FileSet(), modulePath, fc, file, reporter)
	})
}

func (analyzer *Analyzer) checkFile(
	fset *token.FileSet,
	modulePath string,
	fc core.FileChange,
	file *ast.File,
	reporter core.Reporter,
) []core.Finding {
	if len(file.Imports) == 0 {
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

	prevGroup := classify(file.Imports[0].Path.Value)

	for _, spec := range file.Imports[1:] {
		spanGroup := classify(spec.Path.Value)
		if spanGroup < prevGroup {
			return []core.Finding{
				reporter.At(
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
		prevGroup = spanGroup
	}
	return nil
}
