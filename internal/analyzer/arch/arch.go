package arch

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const modulePath = "github.com/0b1-PulsarTech/pulsarules_codex"

var (
	archBoundaryReporter = core.NewReporter("arch-boundary", core.SeverityError, core.CatArch)
	importCycleReporter  = core.NewReporter("import-cycle", core.SeverityError, core.CatArch)
)

// PackageBoundaryAnalyzer checks that inner-layer packages (domain) do not
// import outer-layer packages (infra, transport, cmd).
type PackageBoundaryAnalyzer struct{}

// NewPackageBoundaryAnalyzer creates an analyzer that checks inner-layer packages
// do not import outer-layer packages.
func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{}
}

func (a *PackageBoundaryAnalyzer) ID() string   { return "arch-boundary" }
func (a *PackageBoundaryAnalyzer) Name() string { return "Package boundaries" }
func (a *PackageBoundaryAnalyzer) Description() string {
	return "Checks that inner layers do not depend on outer layers"
}
func (a *PackageBoundaryAnalyzer) Stage() core.StageID     { return core.StageArch }
func (a *PackageBoundaryAnalyzer) Category() core.Category { return core.CatArch }
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
		findings = append(findings, archBoundaryReporter.At(
			".",
			0,
			v,
			"invert the dependency or move the import to the correct layer",
		))
	}
	return findings
}

// ImportCycleAnalyzer detects cycles in the project's import graph.
type ImportCycleAnalyzer struct{}

// NewImportCycleAnalyzer creates an analyzer that detects cycles in the
// project's package import graph.
func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{}
}

func (a *ImportCycleAnalyzer) ID() string   { return "import-cycle" }
func (a *ImportCycleAnalyzer) Name() string { return "Import cycles" }
func (a *ImportCycleAnalyzer) Description() string {
	return "Detects cycles in the package import graph"
}
func (a *ImportCycleAnalyzer) Stage() core.StageID     { return core.StageArch }
func (a *ImportCycleAnalyzer) Category() core.Category { return core.CatArch }
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
		findings = append(findings, importCycleReporter.At(
			".",
			0,
			fmt.Sprintf("import cycle: %s", path),
			"extract the shared dependency into a new package",
		))
	}
	return findings
}
