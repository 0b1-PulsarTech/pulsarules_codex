package packagefilecount

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const analyzerID = "package-file-count"

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter(analyzerID, core.SeverityWarning, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return analyzerID }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.Sources == nil {
		return nil
	}
	params := ctx.Params(analyzerID)
	maxFiles := params.Int("max_files", 10)
	exempt := exemptions{patterns: params.Strings("exempt", nil)}
	eligible := core.ExtensionFilter(ctx, analyzerID, []string{".go"})
	reporter := analyzer.reporter.Resolved(ctx)
	anchors := changedAnchors(ctx, eligible)

	var findings []core.Finding
	if len(ctx.ChangedFiles) > 0 {
		for _, problem := range exempt.invalid() {
			findings = append(findings, reporter.At(
				malformedAnchor(ctx, anchors), 1, problem, "fix the exempt param",
			))
		}
	}
	wanted := make(map[string]bool, len(anchors))
	for _, changed := range anchors {
		wanted[changed.dir] = true
	}
	counts := countByDir(ctx, eligible, wanted)
	for _, changed := range anchors {
		entry := counts[changed.dir]
		if entry.files <= maxFiles || entry.isEntSchema || exempt.matches(changed.dir) {
			continue
		}
		findings = append(findings, reporter.At(
			changed.file,
			1,
			fmt.Sprintf(
				"package %s has %d non-test source files, max %d",
				changed.dir,
				entry.files,
				maxFiles,
			),
			"split the package by responsibility into smaller packages",
		))
	}
	return findings
}
