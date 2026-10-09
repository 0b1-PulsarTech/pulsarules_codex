package importgroup

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter("import-groups", finding.SeverityError, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string             { return "import-groups" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		return checkFile(fc, file, reporter)
	})
}

func checkFile(fc pass.FileChange, file *syntax.File, reporter finding.Reporter) []finding.Finding {
	if len(file.ImportGroups) == 0 || len(file.Imports) == 0 {
		return nil
	}
	prevGroup := file.Imports[0].Group
	for _, spec := range file.Imports[1:] {
		if spec.Group < prevGroup {
			return []finding.Finding{
				reporter.At(
					fc.Path,
					spec.Line,
					fmt.Sprintf(
						"import %s out of order (%s)",
						spec.Literal,
						strings.Join(file.ImportGroups, " → "),
					),
					"reorder imports to follow: "+suggestedOrder(file),
				),
			}
		}
		prevGroup = spec.Group
	}
	return nil
}

func suggestedOrder(file *syntax.File) string {
	names := slices.Clone(file.ImportGroups)
	if file.Language == syntax.LangGo {
		names[len(names)-1] = "this module"
	}
	return strings.Join(names, ", ")
}
