package delegation

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
)

const golangciLintID = "golangci-lint"

// GolangcilintAnalyzer delegates to golangci-lint and converts its output to Findings.
type GolangcilintAnalyzer struct {
	runner *golangcilint.Runner
}

// NewGolangcilintAnalyzer creates an analyzer that delegates to golangci-lint.
func NewGolangcilintAnalyzer(path string) *GolangcilintAnalyzer {
	return &GolangcilintAnalyzer{runner: golangcilint.NewRunner(path)}
}

func (a *GolangcilintAnalyzer) ID() string   { return golangciLintID }
func (a *GolangcilintAnalyzer) Name() string { return "golangci-lint" }
func (a *GolangcilintAnalyzer) Description() string {
	return "Delegates to golangci-lint for lint checks"
}
func (a *GolangcilintAnalyzer) Stage() core.StageID     { return core.StageStatic }
func (a *GolangcilintAnalyzer) Category() core.Category { return core.CatSyntax }
func (a *GolangcilintAnalyzer) Needs() core.Requirements {
	return core.Requirements{}
}

func (a *GolangcilintAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	configPath := extractGolangciConfigPath(ctx)
	return a.runner.Run(ctx.ProjectDir, configPath)
}

// extractGolangciConfigPath reads the optional golangci-config path from the
// analyzer params, returning empty string when not set. It goes through
// ctx.Params like every other analyzer rather than reaching into ctx.Config,
// which dereferences a nil Config.
func extractGolangciConfigPath(ctx *core.AnalysisContext) string {
	return ctx.Params(golangciLintID).String("config_path", "")
}
