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

func (analyzer *Analyzer) ID() string          { return "import-groups" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	modulePath, err := pass.ModulePath(ctx.ProjectDir)
	if err != nil || modulePath == "" {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		return analyzer.checkFile(ctx.ASTCache.FileSet(), modulePath, fc, file, reporter)
	})
}

func (analyzer *Analyzer) checkFile(
	fset *token.FileSet,
	modulePath string,
	fc pass.FileChange,
	file *ast.File,
	reporter pass.Reporter,
) []pass.Finding {
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
			return []pass.Finding{
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
