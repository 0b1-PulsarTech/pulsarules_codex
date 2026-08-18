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

// claudeTarget renders the skills into <base>/.claude/skills, wires the gopls
// MCP into .mcp.json, and wires the reminder hook into the chosen settings file.
type claudeTarget struct{}

var _ Target = claudeTarget{}

func (claudeTarget) Name() string { return "claude" }

// Present reports whether base holds a .claude dir, the one place Install
// writes everything this layout owns (skills, workflows, hooks, settings).
func (claudeTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, hookwire.RootDir))
	return err == nil
}

// Install renders the selected skills under .claude/skills, installs workflows
// composed by those skills under .claude/workflows, then (unless gated off)
// wires the gopls MCP and the reminder hook.
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

// Uninstall removes the rendered skills and workflows (unless ctx.KeepSkills),
// the gopls MCP entry, and the hook wiring Install wrote, reversing Install.
// It unwires every file in ctx.SettingsFiles (both settings files by
// default, since uninstall cannot recover install's --hooks-scope choice),
// then reaps .claude once empty; fsx.RemoveEmptyDir is a no-op otherwise.
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

// unwireClaudeHooks removes the hook wiring from every file in files.
// UnwireSettings filters at the command level, so unwiring a never-wired
// file is a safe no-op, letting this unwire every candidate instead of
// guessing --hooks-scope. Errors fold via errors.Join; the note is gated on
// SettingsChanged, not len(Removed), which also counts gitignore cleanup.
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
		if result.SettingsChanged {
			report.note("removed hook wiring from %s", filepath.Join(claudeDir, settingsFile))
		}
		for _, msg := range result.Restored {
			report.note("%s", msg)
		}
	}
	return errors.Join(errs...)
}
