package noemdash

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter("no-em-dash", finding.SeverityWarning, finding.CatSyntax),
	}
}

func (a *Analyzer) ID() string             { return "no-em-dash" }
func (a *Analyzer) Stage() finding.StageID { return finding.StageStatic }
func (a *Analyzer) Needs() finding.Requirements {
	return finding.Requirements{}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []finding.Finding
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
