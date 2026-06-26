package target

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/opencodewire"
)

// opencodeTarget renders the skills into <base>/.opencode/skills, writes the
// AGENTS.md (carrying the routing contract), wires opencode.json with the
// skills as instructions plus the gopls MCP server, and installs the
// governance plugin that hooks into opencode's event system.
type opencodeTarget struct{}

var _ Target = opencodeTarget{}

// Name is the layout key for the opencode layout.
func (opencodeTarget) Name() string { return "opencode" }

// Install renders the selected skills under .opencode/skills, then writes
// AGENTS.md, wires opencode.json (with the gopls MCP when gopls is present),
// and installs the governance plugin unless NoHooks is set.
func (opencodeTarget) Install(ctx Context) (Report, error) {
	var report Report
	dest := filepath.Join(ctx.Base, opencodewire.SkillsSubdir)
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	gopls := opencodewire.WithoutGopls
	if !ctx.NoMCP {
		if mcpwire.GoplsOnPath() {
			if err := generateGoplsSkill(ctx.Templates, dest); err != nil {
				return report, err
			}
			gopls = opencodewire.WithGopls
		} else {
			report.warn(noGoplsWarning)
		}
	}
	if err := opencodewire.WriteAgents(ctx.Templates, ctx.Base, ctx.Index); err != nil {
		return report, fmt.Errorf("write agents: %w", err)
	}
	if err := opencodewire.WireConfig(ctx.Base, gopls); err != nil {
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
