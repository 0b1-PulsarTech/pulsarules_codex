package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
)

type opencodeTarget struct{}

var _ Target = opencodeTarget{}

func (opencodeTarget) Name() string { return "opencode" }

func (opencodeTarget) Install(ctx Context) (Report, error) {
	var report Report
	dest := filepath.Join(ctx.Base, opencode.SkillsSubdir)
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	gopls := opencode.WithoutGopls
	if !ctx.NoMCP {
		if mcp.GoplsOnPath() {
			if err := generateGoplsSkill(ctx.Templates, dest); err != nil {
				return report, err
			}
			gopls = opencode.WithGopls
		} else {
			report.warn(noGoplsWarning)
		}
	}
	if err := opencode.WriteAgents(ctx.Templates, ctx.Base, ctx.Index); err != nil {
		return report, fmt.Errorf("write agents: %w", err)
	}
	if err := opencode.WireConfig(ctx.Base, gopls); err != nil {
		return report, fmt.Errorf("wire opencode config: %w", err)
	}
	if !ctx.NoHooks {
		if err := ctx.HookInstallers.Install("opencode", ctx.Base, ctx.Templates, ""); err != nil {
			report.warn("opencode plugin: %v", err)
		} else {
			report.note(
				"installed opencode governance plugin: %s",
				filepath.Join(ctx.Base, ".opencode", "plugins"),
			)
		}
	}
	report.note(
		"wired opencode: %s, %s",
		filepath.Join(ctx.Base, ".opencode", "AGENTS.md"),
		filepath.Join(ctx.Base, "opencode.json"),
	)
	return report, nil
}
