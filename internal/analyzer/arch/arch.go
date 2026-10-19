package arch

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func resolveModuleOrFindings(
	reporter finding.Reporter,
	module pass.Module,
) (string, []finding.Finding) {
	if module.Err == nil {
		return module.Path, nil
	}
	// A missing go.mod means no Go project here; any other failure must surface as a finding.
	if errors.Is(module.Err, fs.ErrNotExist) {
		return "", nil
	}
	return "", []finding.Finding{reporter.New(
		fmt.Sprintf("cannot determine the project's module path: %s", module.Err),
	)}
}

type PackageBoundaryAnalyzer struct {
	reporter finding.Reporter
}

func NewPackageBoundaryAnalyzer() *PackageBoundaryAnalyzer {
	return &PackageBoundaryAnalyzer{
		reporter: finding.NewReporter("arch-boundary", finding.SeverityError, finding.CatArch),
	}
}

func (analyzer *PackageBoundaryAnalyzer) ID() string             { return "arch-boundary" }
func (analyzer *PackageBoundaryAnalyzer) Stage() finding.StageID { return finding.StageArch }

func (analyzer *PackageBoundaryAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.GoModule())
	if modulePath == "" {
		return findings
	}

	graph := loadImportGraph(ctx.GoPackages(), modulePath)

	for _, found := range checkBoundaries(graph, modulePath) {
		findings = append(findings, reporter.At(
			found.site.file,
			found.site.line,
			found.message,
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

func (analyzer *ImportCycleAnalyzer) ID() string             { return "import-cycle" }
func (analyzer *ImportCycleAnalyzer) Stage() finding.StageID { return finding.StageArch }

func (analyzer *ImportCycleAnalyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ProjectDir == "" || ctx.Sources == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)

	modulePath, findings := resolveModuleOrFindings(reporter, ctx.GoModule())
	if modulePath == "" {
		return findings
	}

	graph := loadImportGraph(ctx.GoPackages(), modulePath)
	cycles := findCycles(graph.deps)

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
