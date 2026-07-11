package golangcilint

import (
	"os/exec"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var golangciLintReporter = core.NewReporter("golangci-lint", core.SeverityError, core.CatSyntax)

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
		args := []string{"run", "--output.json.path=stdout", modDir + "/..."}
		if configFlag != "" {
			args = append(args, "--config", configFlag)
		}
		//nolint:gosec,noctx
		cmd := exec.Command(r.path, args...)
		cmd.Dir = projectDir
		out, err := cmd.Output()
		allFindings = append(allFindings, parseOutput(out, err)...)
	}
	return allFindings
}

func (r *Runner) runSingle(projectDir, configPath string) []core.Finding {
	args := []string{"run", "--output.json.path=stdout", "./..."}
	configFlag := resolvedConfigFlag(projectDir, configPath)
	if configFlag != "" {
		args = append(args, "--config", configFlag)
	}
	//nolint:gosec,noctx
	cmd := exec.Command(r.path, args...)
	cmd.Dir = projectDir
	out, err := cmd.Output()
	return parseOutput(out, err)
}
