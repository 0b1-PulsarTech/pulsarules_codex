package imports

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

const aliasHint = "configure a project alias (package.json `imports` or tsconfig `paths`) " +
	"and import through it"

type Analyzer struct {
	reporter     core.Reporter
	pathReporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter:     core.NewReporter("import-groups", core.SeverityError, core.CatSyntax),
		pathReporter: core.NewReporter("import-groups/path", core.SeverityWarning, core.CatSyntax),
	}
}

func (analyzer *Analyzer) ID() string          { return "import-groups" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter.Resolved(ctx)
	pathReporter := analyzer.pathReporter.Resolved(ctx)
	return syntaxfiles.Each(ctx, func(fc core.FileChange, file *syntax.File) []core.Finding {
		return append(
			checkFile(fc, file, reporter),
			checkPaths(fc, file, pathReporter)...)
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
					"reorder imports to follow: "+suggestedOrder(file),
				),
			}
		}
		prevGroup = spec.Group
	}
	return nil
}

func checkPaths(fc core.FileChange, file *syntax.File, reporter core.Reporter) []core.Finding {
	var findings []core.Finding
	for _, spec := range file.Imports {
		switch {
		case spec.Prefer != "":
			findings = append(findings, reporter.At(fc.Path, spec.Line,
				fmt.Sprintf("import %s has a better spelling", spec.Literal),
				"write it as "+spec.Prefer))
		case spec.Foreign:
			findings = append(findings, reporter.At(fc.Path, spec.Line,
				fmt.Sprintf("import %s climbs out of its package", spec.Literal),
				"import it through its package name"))
		case spec.Upward:
			findings = append(findings, reporter.At(fc.Path, spec.Line,
				fmt.Sprintf("import %s climbs out of its directory", spec.Literal),
				aliasHint))
		}
	}
	return findings
}

func suggestedOrder(file *syntax.File) string {
	names := slices.Clone(file.ImportGroups)
	if file.Language == syntax.LangGo {
		names[len(names)-1] = "this module"
	}
	return strings.Join(names, ", ")
}
