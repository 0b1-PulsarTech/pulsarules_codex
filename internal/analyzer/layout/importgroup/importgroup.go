package importgroup

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const (
	ID     = "import-groups"
	pathID = ID + "/path"
)

const aliasHint = "configure a project alias (package.json `imports` or tsconfig `paths`) " +
	"and import through it"

type Analyzer struct {
	reporter     finding.Reporter
	pathReporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter:     finding.NewReporter(ID, finding.SeverityError, finding.CatSyntax),
		pathReporter: finding.NewReporter(pathID, finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	pathReporter := pass.ResolveReporter(analyzer.pathReporter, ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		return append(
			checkFile(fc, file, reporter),
			checkPaths(fc, file, pathReporter)...)
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

func checkPaths(
	fc pass.FileChange,
	file *syntax.File,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
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
	if file.Language == language.Go {
		names[len(names)-1] = "this module"
	}
	return strings.Join(names, ", ")
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "imports"}
}
