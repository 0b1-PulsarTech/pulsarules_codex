package arch

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func resolveModuleOrFindings(
	reporter pass.Reporter,
	projectDir string,
) (string, []pass.Finding) {
	modulePath, err := pass.ModulePath(projectDir)
	if err == nil {
		return modulePath, nil
	}
	// A missing go.mod means no Go project here; any other failure must surface as a finding.
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return "", []pass.Finding{reporter.New(
		fmt.Sprintf("cannot determine the project's module path: %s", err),
	)}
}

type PackageBoundaryAnalyzer struct {
	reporter pass.Reporter
}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{
		reporter: pass.NewReporter("arch-boundary", pass.SeverityError, pass.CatArch),
	}
}

func (analyzer *PackageBoundaryAnalyzer) ID() string          { return "arch-boundary" }
func (analyzer *PackageBoundaryAnalyzer) Stage() pass.StageID { return pass.StageArch }

func (analyzer *PackageBoundaryAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.ProjectDir)
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
	reporter pass.Reporter
}

func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{
		reporter: pass.NewReporter("import-cycle", pass.SeverityError, pass.CatArch),
	}
}

func (analyzer *ImportCycleAnalyzer) ID() string          { return "import-cycle" }
func (analyzer *ImportCycleAnalyzer) Stage() pass.StageID { return pass.StageArch }

func (analyzer *ImportCycleAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.ProjectDir)
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
