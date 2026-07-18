package arch

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const modulePath = "github.com/0b1-PulsarTech/pulsarules_codex"

type PackageBoundaryAnalyzer struct {
	reporter finding.Reporter
}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{
		reporter: finding.NewReporter("arch-boundary", finding.SeverityError, finding.CatArch),
	}
}

func (a *PackageBoundaryAnalyzer) ID() string             { return "arch-boundary" }
func (a *PackageBoundaryAnalyzer) Stage() finding.StageID { return finding.StageArch }
func (a *PackageBoundaryAnalyzer) Needs() finding.Requirements {
	return finding.Requirements{}
}

func (a *PackageBoundaryAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	violations := checkBoundaries(idx.graph, modulePath)

	var findings []finding.Finding
	for _, v := range violations {
		findings = append(findings, a.reporter.At(
			".",
			0,
			v,
			"invert the dependency or move the import to the correct layer",
		))
	}
	return findings
}

type ImportCycleAnalyzer struct {
	reporter finding.Reporter
}

func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{
		reporter: finding.NewReporter("import-cycle", finding.SeverityError, finding.CatArch),
	}
}

func (a *ImportCycleAnalyzer) ID() string             { return "import-cycle" }
func (a *ImportCycleAnalyzer) Stage() finding.StageID { return finding.StageArch }
func (a *ImportCycleAnalyzer) Needs() finding.Requirements {
	return finding.Requirements{}
}

func (a *ImportCycleAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	cycles := findCycles(idx.graph)

	var findings []finding.Finding
	for _, cycle := range cycles {
		var b strings.Builder
		for i, p := range cycle {
			if i > 0 {
				b.WriteString(" → ")
			}
			b.WriteString(stripModule(p, modulePath))
		}
		path := b.String()
		findings = append(findings, a.reporter.At(
			".",
			0,
			fmt.Sprintf("import cycle: %s", path),
			"extract the shared dependency into a new package",
		))
	}
	return findings
}
