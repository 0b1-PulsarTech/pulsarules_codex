package arch

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var (
	archBoundaryReporter = core.NewReporter("arch-boundary", core.SeverityError, core.CatArch)
	importCycleReporter  = core.NewReporter("import-cycle", core.SeverityError, core.CatArch)
)

// why: a missing go.mod means "not a Go project here" (same convention as
// the pre-search hook), so it returns no finding, like ctx.ProjectDir == "".
// Any other failure - unreadable/malformed go.mod - is a broken environment:
// silently reporting zero would hide it, the exact defect the hardcoded
// modulePath constant caused - so this returns a Finding instead.
func resolveModulePathOrFindings(
	reporter core.Reporter,
	projectDir string,
) (string, []core.Finding) {
	modulePath, err := core.ModulePath(projectDir)
	if err == nil {
		return modulePath, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return "", []core.Finding{reporter.New(
		fmt.Sprintf("cannot determine the project's module path: %s", err),
	)}
}

// PackageBoundaryAnalyzer checks that inner-layer packages (domain) do not
// import outer-layer packages (infra, transport, cmd).
type PackageBoundaryAnalyzer struct{}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{}
}

func (a *PackageBoundaryAnalyzer) ID() string          { return "arch-boundary" }
func (a *PackageBoundaryAnalyzer) Stage() core.StageID { return core.StageArch }

func (a *PackageBoundaryAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}
	reporter := archBoundaryReporter.Resolved(ctx)

	modulePath, findings := resolveModulePathOrFindings(reporter, ctx.ProjectDir)
	if modulePath == "" {
		return findings
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	violations := checkBoundaries(idx.graph, modulePath)

	for _, v := range violations {
		findings = append(findings, reporter.At(
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

func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{}
}

func (a *ImportCycleAnalyzer) ID() string          { return "import-cycle" }
func (a *ImportCycleAnalyzer) Stage() core.StageID { return core.StageArch }

func (a *ImportCycleAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}
	reporter := importCycleReporter.Resolved(ctx)

	modulePath, findings := resolveModulePathOrFindings(reporter, ctx.ProjectDir)
	if modulePath == "" {
		return findings
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	cycles := findCycles(idx.graph)

	for _, cycle := range cycles {
		var b strings.Builder
		for i, p := range cycle {
			if i > 0 {
				b.WriteString(" → ")
			}
			b.WriteString(stripModule(p, modulePath))
		}
		path := b.String()
		findings = append(findings, reporter.At(
			".",
			0,
			fmt.Sprintf("import cycle: %s", path),
			"extract the shared dependency into a new package",
		))
	}
	return findings
}
