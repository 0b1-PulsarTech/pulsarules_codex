package arch

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const modulePath = "github.com/0b1-PulsarTech/pulsarules_codex"

type PackageBoundaryAnalyzer struct {
	reporter core.Reporter
}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{
		reporter: core.NewReporter("arch-boundary", core.SeverityError, core.CatArch),
	}
}

func (a *PackageBoundaryAnalyzer) ID() string          { return "arch-boundary" }
func (a *PackageBoundaryAnalyzer) Stage() core.StageID { return core.StageArch }
func (a *PackageBoundaryAnalyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *PackageBoundaryAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	violations := checkBoundaries(idx.graph, modulePath)

	var findings []core.Finding
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
	reporter core.Reporter
}

func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{
		reporter: core.NewReporter("import-cycle", core.SeverityError, core.CatArch),
	}
}

func (a *ImportCycleAnalyzer) ID() string          { return "import-cycle" }
func (a *ImportCycleAnalyzer) Stage() core.StageID { return core.StageArch }
func (a *ImportCycleAnalyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *ImportCycleAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	cycles := findCycles(idx.graph)

	var findings []core.Finding
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
