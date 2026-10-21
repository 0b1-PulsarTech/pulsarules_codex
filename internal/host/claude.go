package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

type claudeTarget struct{}

var _ Target = claudeTarget{}

func (claudeTarget) Name() string { return "claude" }

func (claudeTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, claude.RootDir))
	return err == nil
}

func (claudeTarget) Install(ctx Context) (Report, error) {
	var report Report
	claudeDir := filepath.Join(ctx.Base, claude.RootDir)
	dest := filepath.Join(claudeDir, claude.SkillsSubdir)
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	workflowDest := filepath.Join(claudeDir, "workflows")
	if err := installWorkflows(ctx, workflowDest, &report); err != nil {
		return report, err
	}
	if !ctx.NoMCP {
		if err := wireClaudeMCP(ctx, dest, &report); err != nil {
			return report, err
		}
	}
	if ctx.NoHooks {
		return report, nil
	}
	sub, err := install.Claude(install.Context{
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
	claudeDir := filepath.Join(ctx.Base, claude.RootDir)
	if !ctx.KeepSkills {
		if err := removeSkills(
			filepath.Join(claudeDir, claude.SkillsSubdir),
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
	if err := unwireClaudeHooks(claudeDir, ctx.SettingsFiles, &report); err != nil {
		return report, err
	}
	if err := disk.RemoveEmptyDir(claudeDir); err != nil {
		return report, fmt.Errorf("remove empty claude dir: %w", err)
	}
	return report, nil
}

func unwireClaudeHooks(claudeDir string, files []string, report *Report) error {
	var errs []error
	for _, settingsFile := range files {
		uctx := install.UninstallContext{Dir: claudeDir, SettingsFile: settingsFile}
		sub, err := install.UninstallClaude(uctx)
		if err != nil {
			if errors.Is(err, jsondoc.ErrUnparseable) {
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
