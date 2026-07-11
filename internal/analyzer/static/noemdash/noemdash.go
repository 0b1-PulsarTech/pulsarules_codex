package noemdash

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports any Go source file that contains an em-dash character
// (U+2014). The project style guide requires hyphens instead.
type Analyzer struct{}

var noEmDashReporter = core.NewReporter("no-em-dash", core.SeverityWarning, core.CatSyntax)

// NewAnalyzer creates a no-em-dash analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "no-em-dash" }
func (a *Analyzer) Name() string { return "No em-dash" }
func (a *Analyzer) Description() string {
	return "Reports em-dash characters (U+2014) in Go files; use a hyphen instead"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageStatic }
func (a *Analyzer) Category() core.Category { return core.CatSyntax }
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
				findings = append(findings, noEmDashReporter.At(
					fc.Path,
					i+1,
					"em-dash (--) found; use a hyphen (-) instead",
					`replace "--" with "-"`,
				))
			}
		}
	}
	return findings
}
