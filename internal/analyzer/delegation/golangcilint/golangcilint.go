package golangcilint

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func forcedLinters() []string {
	// Forced via -E on every delegated run: against silence, not an explicit opt-out.
	return []string{
		"nolintlint",
		"paralleltest",
		"tparallel",
		"thelper",
		"forcetypeassert",
		"nilerr",
	}
}

type Runner struct {
	path    string
	timeout time.Duration
	failure finding.Reporter
	warning finding.Reporter
}

func NewRunner(golangciLintPath string) *Runner {
	const golangciLintTimeout = 2 * time.Minute
	return &Runner{
		path: golangciLintPath,
		// Per call, not aggregated over runPerModule's loop: modules vary in size,
		// so sharing one budget would starve a large one.
		timeout: golangciLintTimeout,
		failure: finding.NewReporter("golangci-lint", finding.SeverityError, finding.CatSyntax),
		warning: finding.NewReporter("golangci-lint", finding.SeverityWarning, finding.CatSyntax),
	}
}

func (runner *Runner) Run(projectDir, configPath string) []finding.Finding {
	if runner.path == "" {
		return nil
	}
	// A missing binary is an environment fact, not a finding.
	if _, err := exec.LookPath(runner.path); err != nil {
		return nil
	}
	return runner.runLint(projectDir, configPath)
}

func (runner *Runner) runLint(projectDir, configPath string) []finding.Finding {
	if hasGoWork(projectDir) {
		return runner.runPerModule(projectDir, configPath)
	}

	return runner.runSingle(projectDir, configPath)
}

func (runner *Runner) runPerModule(projectDir, configPath string) []finding.Finding {
	modules := parseGoWorkModules(projectDir)
	if len(modules) == 0 {
		return runner.runSingle(projectDir, configPath)
	}

	configFlag := resolvedConfigFlag(projectDir, configPath)

	allFindings := runner.forcedBaselineFindings(configFlag)
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		result, err := runner.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, runner.parseOutput(result, err)...)
	}
	return allFindings
}

func (runner *Runner) runSingle(projectDir, configPath string) []finding.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	result, err := runner.run(projectDir, lintArgs("./...", configFlag))
	return append(runner.forcedBaselineFindings(configFlag), runner.parseOutput(result, err)...)
}

func (runner *Runner) run(projectDir string, args []string) (process.Result, error) {
	result, err := command.Execute(process.Command{
		Name:    runner.path,
		Args:    args,
		Dir:     projectDir,
		Timeout: runner.timeout,
	})
	if err != nil {
		// %w keeps *execx.Error (and the *exec.ExitError it wraps) reachable through
		// command.Classify.
		return result, fmt.Errorf("run golangci-lint: %w", err)
	}
	return result, nil
}

func lintArgs(target, configFlag string) []string {
	args := []string{"run", "--output.json.path=stdout"}
	// -E adds the forced linters on top of the target's own config; it never replaces it.
	for _, linter := range forcedLinters() {
		args = append(args, "-E", linter)
	}
	if configFlag != "" {
		args = append(args, "--config", configFlag)
	}
	return append(args, target)
}
