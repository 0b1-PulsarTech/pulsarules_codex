package imports

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer checks that Go import statements follow the three-group
// convention (stdlib, external, this module), each a contiguous block.
//
// why: it holds no module path. It used to be hardcoded to this tool's own
// module, so any other project's "this module" group matched nothing.
type Analyzer struct{}

var importGroupsReporter = core.NewReporter("import-groups", core.SeverityError, core.CatSyntax)

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string          { return "import-groups" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

// Analyze reports every file whose import blocks break the three-group order.
//
// why: an unreadable go.mod means "this module" is unknowable, so it stays
// silent rather than guess - arch-boundary already reports the broken-go.mod
// case, and a second finding would be noise.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	modulePath, err := core.ModulePath(ctx.ProjectDir)
	if err != nil || modulePath == "" {
		return nil
	}
	reporter := importGroupsReporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), modulePath, fc, f, reporter)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	modulePath string,
	fc core.FileChange,
	f *ast.File,
	reporter core.Reporter,
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
		prevGroup = g
	}
	return nil
}
