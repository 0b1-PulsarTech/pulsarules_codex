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

func (a *PackageBoundaryAnalyzer) ID() string          { return "arch-boundary" }
func (a *PackageBoundaryAnalyzer) Stage() pass.StageID { return pass.StageArch }

func (a *PackageBoundaryAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}
	reporter := pass.ResolveReporter(a.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.ProjectDir)
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

type ImportCycleAnalyzer struct {
	reporter pass.Reporter
}

func NewImportCycleAnalyzer() *ImportCycleAnalyzer {
	return &ImportCycleAnalyzer{
		reporter: pass.NewReporter("import-cycle", pass.SeverityError, pass.CatArch),
	}
}

func (a *ImportCycleAnalyzer) ID() string          { return "import-cycle" }
func (a *ImportCycleAnalyzer) Stage() pass.StageID { return pass.StageArch }

func (a *ImportCycleAnalyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if ctx.ProjectDir == "" {
		return nil
	}
	reporter := pass.ResolveReporter(a.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.ProjectDir)
	if modulePath == "" {
		return findings
	}

	idx := loadProjectIndex(ctx.ProjectDir, modulePath)
	cycles := findCycles(idx.graph)

	for _, cycle := range cycles {
		var b strings.Builder
		for hop, p := range cycle {
			if hop > 0 {
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
