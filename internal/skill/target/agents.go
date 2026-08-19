package target

import (
	"os"
	"path/filepath"
)

type agentsTarget struct{}

var _ Target = agentsTarget{}

func (agentsTarget) Name() string { return "agents" }

func (agentsTarget) Present(base string) bool {
	if _, err := os.Stat(filepath.Join(base, "AGENTS.md")); err != nil {
		return false
	}
	// Defer to opencode, which also writes and reverses the root AGENTS.md.
	return !opencodeTarget{}.Present(base)
}

func (agentsTarget) Install(ctx Context) (Report, error) {
	var report Report
	if err := writeAgents(ctx, &report); err != nil {
		return report, err
	}
	return report, nil
}

func (agentsTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	if ctx.KeepSkills {
		return report, nil
	}
	if err := removeAgents(ctx.Base, &report); err != nil {
		return report, err
	}
	return report, nil
}
