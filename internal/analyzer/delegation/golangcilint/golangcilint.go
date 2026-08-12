package golangcilint

import (
	"os/exec"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Runner struct {
	path    string
	failure core.Reporter
}

func NewRunner(golangciLintPath string) *Runner {
	return &Runner{
		path:    golangciLintPath,
		failure: core.NewReporter("golangci-lint", core.SeverityError, core.CatSyntax),
	}
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
		allFindings = append(allFindings, r.parseOutput(out, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []core.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	out, err := r.run(projectDir, lintArgs("./...", configFlag))
	return r.parseOutput(out, err)
}

func (r *Runner) run(projectDir string, args []string) ([]byte, error) {
	//nolint:gosec,noctx
	cmd := exec.Command(r.path, args...)
	cmd.Dir = projectDir
	return cmd.Output()
}

func lintArgs(target, configFlag string) []string {
	args := []string{"run", "--output.json.path=stdout"}
	if configFlag != "" {
		args = append(args, "--config", configFlag)
	}
	return append(args, target)
}
