package imports

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	modulePath string
	reporter   core.Reporter
}

func NewAnalyzer(modulePath string) *Analyzer {
	return &Analyzer{
		modulePath: modulePath,
		reporter:   core.NewReporter("import-groups", core.SeverityError, core.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "import-groups" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.modulePath == "" || ctx.ASTCache == nil {
		return nil
	}

	fset := ctx.ASTCache.FileSet()
	var findings []core.Finding
	for fc, f := range ctx.ChangedGoASTs() {
		findings = append(findings, a.checkFile(fset, fc, f)...)
	}
	return findings
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
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
		if a.modulePath != "" &&
			(clean == a.modulePath || strings.HasPrefix(clean, a.modulePath+"/")) {
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
						a.modulePath,
					),
					"reorder imports to follow: stdlib, external, this module",
				),
			}
		}
		prevGroup = g
	}
	return nil
}
