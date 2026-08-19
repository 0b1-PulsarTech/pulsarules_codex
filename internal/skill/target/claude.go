package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
)

type claudeTarget struct{}

var _ Target = claudeTarget{}

func (claudeTarget) Name() string { return "claude" }

func (claudeTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, hookwire.RootDir))
	return err == nil
}

func (claudeTarget) Install(ctx Context) (Report, error) {
	var report Report
	claudeDir := filepath.Join(ctx.Base, hookwire.RootDir)
	dest := filepath.Join(claudeDir, hookwire.SkillsSubdir)
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	workflowDest := filepath.Join(claudeDir, "workflows")
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
	sub, err := ctx.HookInstallers.Install("claude", install.Context{
		Dir:          claudeDir,
		Templates:    ctx.Templates,
		SettingsFile: ctx.SettingsFile,
	})
	report.Merge(sub)
	if err != nil {
		return report, fmt.Errorf("install hooks: %w", err)
	}
	return report, nil
}

func (claudeTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	claudeDir := filepath.Join(ctx.Base, hookwire.RootDir)
	if !ctx.KeepSkills {
		if err := removeSkills(
			filepath.Join(claudeDir, hookwire.SkillsSubdir),
			&report,
		); err != nil {
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
		sub, err := hooks.Uninstall("claude", uctx)
		if err != nil {
			if errors.Is(err, fsx.ErrUnparseableJSON) {
				report.Warn("%v", err)
				continue
			}
			errs = append(errs, fmt.Errorf("uninstall hooks (%s): %w", settingsFile, err))
			continue
		}
		report.Merge(sub)
	}
	return errors.Join(errs...)
}
