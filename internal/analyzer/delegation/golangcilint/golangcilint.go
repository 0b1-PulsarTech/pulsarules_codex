package golangcilint

import (
	"fmt"
	"path/filepath"

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

func (analyzer *Analyzer) runLint(projectDir, configPath string) []finding.Finding {
	if hasGoWork(projectDir) {
		return analyzer.runPerModule(projectDir, configPath)
	}

	return analyzer.runSingle(projectDir, configPath)
}

func (analyzer *Analyzer) runPerModule(projectDir, configPath string) []finding.Finding {
	modules := parseGoWorkModules(projectDir)
	if len(modules) == 0 {
		return analyzer.runSingle(projectDir, configPath)
	}

	configFlag := resolvedConfigFlag(projectDir, configPath)

	allFindings := analyzer.forcedBaselineFindings(configFlag)
	for _, mod := range modules {
		modDir := filepath.Join(projectDir, mod)
		result, err := analyzer.run(projectDir, lintArgs(modDir+"/...", configFlag))
		allFindings = append(allFindings, analyzer.parseOutput(result, err)...)
	}
	return allFindings
}

func (analyzer *Analyzer) runSingle(projectDir, configPath string) []finding.Finding {
	configFlag := resolvedConfigFlag(projectDir, configPath)
	result, err := analyzer.run(projectDir, lintArgs("./...", configFlag))
	return append(analyzer.forcedBaselineFindings(configFlag), analyzer.parseOutput(result, err)...)
}

func (analyzer *Analyzer) run(projectDir string, args []string) (process.Result, error) {
	result, err := command.Execute(process.Command{
		Name:    analyzer.path,
		Args:    args,
		Dir:     projectDir,
		Timeout: analyzer.timeout,
	})
	if err != nil {
		// %w keeps *process.Error (and the *exec.ExitError it wraps) reachable through
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
