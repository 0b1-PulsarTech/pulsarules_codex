package arch

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func resolveModulePathOrFindings(
	reporter core.Reporter,
	projectDir string,
) (string, []core.Finding) {
	modulePath, err := core.ModulePath(projectDir)
	if err == nil {
		return modulePath, nil
	}
	// A missing go.mod means no Go project here; any other failure must surface as a finding.
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return "", []core.Finding{reporter.New(
		fmt.Sprintf("cannot determine the project's module path: %s", err),
	)}
}

type PackageBoundaryAnalyzer struct {
	reporter core.Reporter
}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{
		reporter: core.NewReporter("arch-boundary", core.SeverityError, core.CatArch),
	}
}

func (analyzer *PackageBoundaryAnalyzer) ID() string          { return "arch-boundary" }
func (analyzer *PackageBoundaryAnalyzer) Stage() core.StageID { return core.StageArch }

func (analyzer *PackageBoundaryAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := analyzer.reporter.Resolved(ctx)

	modulePath, findings := resolveModulePathOrFindings(reporter, ctx.ProjectDir)
	if modulePath == "" {
		return findings
	}

	idx := loadProjectIndex(ctx.Sources, modulePath)
	violations := checkBoundaries(idx.graph, modulePath)

	for _, violation := range violations {
		findings = append(findings, reporter.At(
			".",
			0,
			violation,
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

func (analyzer *ImportCycleAnalyzer) ID() string          { return "import-cycle" }
func (analyzer *ImportCycleAnalyzer) Stage() core.StageID { return core.StageArch }

func (analyzer *ImportCycleAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := analyzer.reporter.Resolved(ctx)

	modulePath, findings := resolveModulePathOrFindings(reporter, ctx.ProjectDir)
	if modulePath == "" {
		return findings
	}

	idx := loadProjectIndex(ctx.Sources, modulePath)
	cycles := findCycles(idx.graph)

	for _, cycle := range cycles {
		var builder strings.Builder
		for hop, pkg := range cycle {
			if hop > 0 {
				builder.WriteString(" → ")
			}
			builder.WriteString(stripModule(pkg, modulePath))
		}
		path := builder.String()
		findings = append(findings, reporter.At(
			".",
			0,
			fmt.Sprintf("import cycle: %s", path),
			"extract the shared dependency into a new package",
		))
	}
	return findings
}
