package importgroup

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
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
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []pass.Finding {
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc pass.FileChange, file *syntax.File, reporter pass.Reporter) []pass.Finding {
	if len(file.ImportGroups) == 0 || len(file.Imports) == 0 {
		return nil
	}
	prevGroup := file.Imports[0].Group
	for _, spec := range file.Imports[1:] {
		if spec.Group < prevGroup {
			return []pass.Finding{
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
