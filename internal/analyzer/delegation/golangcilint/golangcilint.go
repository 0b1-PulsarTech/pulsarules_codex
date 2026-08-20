package golangcilint

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

// golangciLintTimeout bounds one golangci-lint invocation (single-module,
// or one call per module in runPerModule's loop for a go.work repo).
// why: applied per-call, not aggregated over that loop - modules vary in
// size, so sharing one budget would starve a large one.
const golangciLintTimeout = 2 * time.Minute

var golangciLintReporter = core.NewReporter("golangci-lint", core.SeverityError, core.CatSyntax)

// why: the baseline forced via -E on every delegated run, on top of whatever the target's own
// config enables (see knowledge/standards/rules/infra/build.md). It is forced against SILENCE, not
// against an explicit opt-out - see forcedBaselineFindings for the one case -E loses.
var forcedLinters = []string{
	"nolintlint",
	"paralleltest",
	"tparallel",
	"thelper",
	"forcetypeassert",
	"nilerr",
}

type Runner struct {
	path    string
	timeout time.Duration
}

func NewRunner(golangciLintPath string) *Runner {
	return &Runner{path: golangciLintPath, timeout: golangciLintTimeout}
}

// Run reports lint findings, or nil when golangci-lint is unavailable: an
// empty path, or a path that does not resolve to an executable (mirroring
// mcpwire.GoplsOnPath's probe). A missing binary is an environment fact,
// not a finding - it used to always spawn and surface as a synthetic one.
func (r *Runner) Run(projectDir, configPath string) []core.Finding {
	if r.path == "" {
		return nil
	}
	if _, err := exec.LookPath(r.path); err != nil {
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

	allFindings := forcedBaselineFindings(configFlag)
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		result, err := r.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, parseOutput(result, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []core.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	result, err := r.run(projectDir, lintArgs("./...", configFlag))
	return append(forcedBaselineFindings(configFlag), parseOutput(result, err)...)
}

// why: %w keeps *execx.Error (and the *exec.ExitError it wraps) reachable
// through parseOutput's errors.As.
func (r *Runner) run(projectDir string, args []string) (execx.Result, error) {
	result, err := execx.Run(context.Background(), execx.Command{
		Name:    r.path,
		Args:    args,
		Dir:     projectDir,
		Timeout: r.timeout,
	})
	if err != nil {
		return result, fmt.Errorf("run golangci-lint: %w", err)
	}
	return result, nil
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
