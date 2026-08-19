package golangcilint

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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
	failure pass.Reporter
	warning pass.Reporter
}

func NewRunner(golangciLintPath string) *Runner {
	return &Runner{
		path:    golangciLintPath,
		failure: pass.NewReporter("golangci-lint", pass.SeverityError, pass.CatSyntax),
		warning: pass.NewReporter("golangci-lint", pass.SeverityWarning, pass.CatSyntax),
	}
}

func (r *Runner) Run(projectDir, configPath string) []pass.Finding {
	if r.path == "" {
		return nil
	}
	return r.runLint(projectDir, configPath)
}

func (r *Runner) runLint(projectDir, configPath string) []pass.Finding {
	if hasGoWork(projectDir) {
		return r.runPerModule(projectDir, configPath)
	}

	return r.runSingle(projectDir, configPath)
}

func (r *Runner) runPerModule(projectDir, configPath string) []pass.Finding {
	modules := parseGoWorkModules(projectDir)
	if len(modules) == 0 {
		return r.runSingle(projectDir, configPath)
	}

	configFlag := resolvedConfigFlag(projectDir, configPath)

	allFindings := r.forcedBaselineFindings(configFlag)
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		result, err := r.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, r.parseOutput(result, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []pass.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	result, err := r.run(projectDir, lintArgs("./...", configFlag))
	return append(r.forcedBaselineFindings(configFlag), r.parseOutput(result, err)...)
}

func (r *Runner) run(projectDir string, args []string) (process.Result, error) {
	result, err := process.Run(context.Background(), process.Command{
		Name: r.path,
		Args: args,
		Dir:  projectDir,
	})
	if err != nil {
		// %w keeps *execx.Error (and the *exec.ExitError it wraps) reachable through
		// parseOutput's errors.As.
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
