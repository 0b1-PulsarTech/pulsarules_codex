package golangcilint

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
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
	failure finding.Reporter
	warning finding.Reporter
}

func NewRunner(golangciLintPath string) *Runner {
	return &Runner{
		path:    golangciLintPath,
		failure: finding.NewReporter("golangci-lint", finding.SeverityError, finding.CatSyntax),
		warning: finding.NewReporter("golangci-lint", finding.SeverityWarning, finding.CatSyntax),
	}
}

func (r *Runner) Run(projectDir, configPath string) []finding.Finding {
	if r.path == "" {
		return nil
	}
	return r.runLint(projectDir, configPath)
}

func (r *Runner) runLint(projectDir, configPath string) []finding.Finding {
	if hasGoWork(projectDir) {
		return r.runPerModule(projectDir, configPath)
	}

	return r.runSingle(projectDir, configPath)
}

func (r *Runner) runPerModule(projectDir, configPath string) []finding.Finding {
	modules := parseGoWorkModules(projectDir)
	if len(modules) == 0 {
		return r.runSingle(projectDir, configPath)
	}

	configFlag := resolvedConfigFlag(projectDir, configPath)

	allFindings := r.forcedBaselineFindings(configFlag)
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		out, err := r.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, r.parseOutput(out, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []finding.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	out, err := r.run(projectDir, lintArgs("./...", configFlag))
	return append(r.forcedBaselineFindings(configFlag), r.parseOutput(out, err)...)
}

func (r *Runner) run(projectDir string, args []string) ([]byte, error) {
	//nolint:gosec,noctx // r.path is the configured golangci-lint binary, and Run has no context.
	cmd := exec.Command(r.path, args...)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("run golangci-lint: %w", err)
	}
	return out, nil
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
