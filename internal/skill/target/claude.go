package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

type claudeTarget struct{}

var _ Target = claudeTarget{}

func (claudeTarget) Name() string { return "claude" }

func (claudeTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, ".claude"))
	return err == nil
}

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

func (claudeTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	claudeDir := filepath.Join(ctx.Base, ".claude")
	if !ctx.KeepSkills {
		if err := removeSkills(filepath.Join(claudeDir, "skills"), &report); err != nil {
			return report, err
		}
		if err := removeWorkflows(filepath.Join(claudeDir, "workflows"), &report); err != nil {
			return report, err
		}
	}
	if err := unwireClaudeMCP(ctx.Base, &report); err != nil {
		return report, err
	}
	if err := unwireClaudeHooks(
		ctx.HookUninstallers,
		claudeDir,
		ctx.SettingsFiles,
		&report,
	); err != nil {
		return report, err
	}
	if err := fsx.RemoveEmptyDir(claudeDir); err != nil {
		return report, fmt.Errorf("remove empty claude dir: %w", err)
	}
	return report, nil
}

func unwireClaudeHooks(
	hooks *install.Registry, claudeDir string, files []string, report *Report,
) error {
	var errs []error
	for _, settingsFile := range files {
		uctx := install.UninstallContext{Dir: claudeDir, SettingsFile: settingsFile}
		result, err := hooks.Uninstall("claude", uctx)
		if err != nil {
			if errors.Is(err, fsx.ErrUnparseableJSON) {
				report.warn("%v", err)
				continue
			}
			errs = append(errs, fmt.Errorf("uninstall hooks (%s): %w", settingsFile, err))
			continue
		}
		if len(result.Removed) > 0 {
			report.note("removed hook wiring from %s", filepath.Join(claudeDir, settingsFile))
		}
		for _, msg := range result.Restored {
			report.note("%s", msg)
		}
	}
	return errors.Join(errs...)
}
