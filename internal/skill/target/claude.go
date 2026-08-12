package target

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
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
	}); err != nil {
		return report, fmt.Errorf("install hooks: %w", err)
	}
	report.note("wired hook into %s", filepath.Join(claudeDir, ctx.SettingsFile))
	return report, nil
}

func wireClaudeMCP(templates fs.FS, repoDir, skillsDir string, report *Report) error {
	if !mcpwire.GoplsOnPath() {
		report.warn(noGoplsWarning)
		return nil
	}
	if err := mcpwire.WriteMCP(templates, repoDir); err != nil {
		return fmt.Errorf("write mcp: %w", err)
	}
	if err := generateGoplsSkill(templates, skillsDir); err != nil {
		return fmt.Errorf("generate gopls skill: %w", err)
	}
	report.note(
		"wired gopls MCP into %s; generated gopls-navigation skill",
		filepath.Join(repoDir, ".mcp.json"),
	)
	return nil
}

func generateGoplsSkill(templates fs.FS, skillsDir string) error {
	const goplsInstructionsTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), goplsInstructionsTimeout)
	defer cancel()
	instructions, err := mcpwire.GoplsInstructions(ctx)
	if err != nil {
		return fmt.Errorf("get gopls instructions: %w", err)
	}
	if genErr := mcpwire.GenerateGoplsSkill(templates, skillsDir, instructions); genErr != nil {
		return fmt.Errorf("generate gopls skill: %w", genErr)
	}
	return nil
}
