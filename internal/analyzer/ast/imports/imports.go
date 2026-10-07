package imports

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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
	reporter := analyzer.reporter.Resolved(ctx)
	return syntaxfiles.Each(ctx, func(fc core.FileChange, file *syntax.File) []core.Finding {
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc core.FileChange, file *syntax.File, reporter core.Reporter) []core.Finding {
	if len(file.ImportGroups) == 0 || len(file.Imports) == 0 {
		return nil
	}
	prevGroup := file.Imports[0].Group
	for _, spec := range file.Imports[1:] {
		if spec.Group < prevGroup {
			return []core.Finding{
				reporter.At(
					fc.Path,
					spec.Line,
					fmt.Sprintf(
						"import %s out of order (%s)",
						spec.Literal,
						strings.Join(file.ImportGroups, " → "),
					),
					"reorder imports to follow: "+suggestedOrder(file.ImportGroups),
				),
			}
		}
		prevGroup = spec.Group
	}
	return nil
}

func suggestedOrder(groups []string) string {
	names := slices.Clone(groups)
	names[len(names)-1] = "this module"
	return strings.Join(names, ", ")
}
