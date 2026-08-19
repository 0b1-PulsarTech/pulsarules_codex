package golangcilint

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

const golangciLintID = "golangci-lint"

type Analyzer struct {
	runner *Runner
}

func NewAnalyzer(path string) *Analyzer {
	return &Analyzer{runner: NewRunner(path)}
}

func (a *Analyzer) ID() string          { return golangciLintID }
func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	configPath := extractGolangciConfigPath(ctx)
	return a.runner.Run(ctx.ProjectDir, configPath)
}

func extractGolangciConfigPath(ctx *pass.Context) string {
	// Through ctx.Params, not ctx.Config, which dereferences a nil Config.
	return ctx.Params(golangciLintID).String("config_path", "")
}
