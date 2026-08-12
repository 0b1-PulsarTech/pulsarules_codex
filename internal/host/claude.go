package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
)

type claudeTarget struct{}

var _ Target = claudeTarget{}

func (claudeTarget) Name() string { return "claude" }

func (claudeTarget) Install(ctx Context) (Report, error) {
	var report Report
	dest := filepath.Join(ctx.Base, ".claude", "skills")
	claudeDir := filepath.Join(ctx.Base, ".claude")
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	workflowDest := filepath.Join(ctx.Base, ".claude", "workflows")
	if err := installWorkflows(ctx, workflowDest, &report); err != nil {
		return report, err
	}
	if !ctx.NoMCP {
		if err := wireClaudeMCP(ctx.Templates, ctx.Base, dest, &report); err != nil {
			return report, err
		}
	}
	if ctx.NoHooks {
		return report, nil
	}
	if err := ctx.HookInstallers.Install("claude", install.Context{
		Dir:          claudeDir,
		Templates:    ctx.Templates,
		SettingsFile: ctx.SettingsFile,
		Warn:         report.warn,
	}); err != nil {
		return report, fmt.Errorf("install hooks: %w", err)
	}
	report.note("wired hook into %s", filepath.Join(claudeDir, ctx.SettingsFile))
	return report, nil
}
