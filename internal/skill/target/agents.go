package target

import (
	"os"
	"path/filepath"
)

// agentsTarget writes only the project-root AGENTS.md, for the AI coding
// agents that read a repo-root AGENTS.md and nothing else (Codex, Gemini
// CLI, Zed, Amp, JetBrains Junie, Jules, and others). It renders no skill
// docs and wires no config; a project installing this alongside opencode
// still gets one identical file, since both call agentswire.WriteAgents.
type agentsTarget struct{}

var _ Target = agentsTarget{}

func (agentsTarget) Name() string { return "agents" }

// Present reports whether base holds an AGENTS.md this layout could have written on its own.
// opencodeTarget also writes the same root AGENTS.md as a side effect and already reverses it
// from its own Uninstall, so Present defers to it here: without this, DetectTargets would return
// both "agents" and "opencode" for an opencode-only project, and both Uninstalls would attempt
// removeAgents.
func (agentsTarget) Present(base string) bool {
	if _, err := os.Stat(filepath.Join(base, "AGENTS.md")); err != nil {
		return false
	}
	return !opencodeTarget{}.Present(base)
}

// Install renders AGENTS.md at ctx.Base, listing the selected skills.
func (agentsTarget) Install(ctx Context) (Report, error) {
	var report Report
	if err := writeAgents(ctx, &report); err != nil {
		return report, err
	}
	return report, nil
}

// Uninstall removes the AGENTS.md Install wrote, unless ctx.KeepSkills is
// set or the file's content proves it was never this tool's - a
// user-authored root AGENTS.md survives untouched (see removeAgents).
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
