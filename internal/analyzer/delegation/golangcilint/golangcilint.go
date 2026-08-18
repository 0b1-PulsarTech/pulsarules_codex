package golangcilint

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var golangciLintReporter = core.NewReporter("golangci-lint", core.SeverityError, core.CatSyntax)

// why: the non-negotiable baseline forced via -E on every delegated run, on top of whatever the
// target's own config enables (see knowledge/standards/rules/infra/build.md).
var forcedLinters = []string{
	"nolintlint",
	"paralleltest",
	"tparallel",
	"thelper",
	"forcetypeassert",
	"nilerr",
}

type Runner struct {
	path string
}

func NewRunner(golangciLintPath string) *Runner {
	return &Runner{path: golangciLintPath}
}

func (r *Runner) Run(projectDir, configPath string) []core.Finding {
	if r.path == "" {
		return nil
	}
	return r.runLint(projectDir, configPath)
}

func (r *Runner) runLint(projectDir, configPath string) []core.Finding {
	if hasGoWork(projectDir) {
		return r.runPerModule(projectDir, configPath)
	}

	return r.runSingle(projectDir, configPath)
}

func (r *Runner) runPerModule(projectDir, configPath string) []core.Finding {
	modules := parseGoWorkModules(projectDir)
	if len(modules) == 0 {
		return r.runSingle(projectDir, configPath)
	}

	configFlag := resolvedConfigFlag(projectDir, configPath)

	var allFindings []core.Finding
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		out, err := r.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, parseOutput(out, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []core.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	out, err := r.run(projectDir, lintArgs("./...", configFlag))
	return parseOutput(out, err)
}

// why: %w keeps *exec.ExitError reachable through parseOutput's errors.As.
func (r *Runner) run(projectDir string, args []string) ([]byte, error) {
	//nolint:gosec,noctx // args are our own built flags/paths, not user input; no per-call timeout by design, run.timeout in the target's config governs.
	cmd := exec.Command(r.path, args...)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("run golangci-lint: %w", err)
	}
	return out, nil
}

// why: -E adds forcedLinters on top of the target's own config; it never replaces it.
func lintArgs(target, configFlag string) []string {
	args := []string{"run", "--output.json.path=stdout"}
	for _, linter := range forcedLinters {
		args = append(args, "-E", linter)
	}
	if configFlag != "" {
		args = append(args, "--config", configFlag)
	}
	return append(args, target)
}
