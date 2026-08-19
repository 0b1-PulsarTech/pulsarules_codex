package delegation

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/golangcilint"
)

const golangciLintID = "golangci-lint"

type GolangcilintAnalyzer struct {
	runner *golangcilint.Runner
}

func NewGolangcilintAnalyzer(path string) *GolangcilintAnalyzer {
	return &GolangcilintAnalyzer{runner: golangcilint.NewRunner(path)}
}

func (a *GolangcilintAnalyzer) ID() string          { return golangciLintID }
func (a *GolangcilintAnalyzer) Stage() core.StageID { return core.StageStatic }

func (a *GolangcilintAnalyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	configPath := extractGolangciConfigPath(ctx)
	return a.runner.Run(ctx.ProjectDir, configPath)
}

func extractGolangciConfigPath(ctx *core.AnalysisContext) string {
	// Through ctx.Params, not ctx.Config, which dereferences a nil Config.
	return ctx.Params(golangciLintID).String("config_path", "")
}
