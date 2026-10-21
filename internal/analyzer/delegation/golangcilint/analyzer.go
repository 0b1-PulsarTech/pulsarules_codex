package golangcilint

import (
	"os/exec"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	ID              = "golangci-lint"
	paramConfigPath = "config_path"
)

type Analyzer struct {
	path    string
	timeout time.Duration
	failure finding.Reporter
	warning finding.Reporter
}

func NewAnalyzer(golangciLintPath string) *Analyzer {
	const golangciLintTimeout = 2 * time.Minute
	return &Analyzer{
		path: golangciLintPath,
		// Per call, not aggregated over runPerModule's loop: modules vary in size,
		// so sharing one budget would starve a large one.
		timeout: golangciLintTimeout,
		failure: finding.NewReporter(ID, finding.SeverityError, finding.CatSyntax),
		warning: finding.NewReporter(ID, finding.SeverityWarning, finding.CatSyntax),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if analyzer.path == "" {
		return nil
	}
	// A missing binary is an environment fact, not a finding.
	if _, err := exec.LookPath(analyzer.path); err != nil {
		return nil
	}
	return analyzer.runLint(ctx.ProjectDir, extractGolangciConfigPath(ctx))
}

func extractGolangciConfigPath(ctx *pass.Context) string {
	// Through ctx.Params, not ctx.Config, which dereferences a nil Config.
	return ctx.Params(ID).String(paramConfigPath, "")
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "code-smells",
		Params: []string{paramConfigPath},
	}
}
