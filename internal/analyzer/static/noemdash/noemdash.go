package noemdash

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter("no-em-dash", core.SeverityWarning, core.CatSyntax),
	}
}

func (a *Analyzer) ID() string          { return "no-em-dash" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.ContainsRune(line, '\u2014') {
				findings = append(findings, a.reporter.At(
					fc.Path,
					i+1,
					"em dash (U+2014) found; use a hyphen (-) instead",
					"replace the em dash with -",
				))
			}
		}
	}
	return findings
}
