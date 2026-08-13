package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/opencodewire"
)

// opencodeTarget renders the skills into <base>/.opencode/skills, writes the
// root AGENTS.md (carrying the routing contract, via the same builder the
// thin agents target uses), wires opencode.json with the skills as
// instructions plus the gopls MCP server, and installs the governance plugin
// that hooks into opencode's event system.
type opencodeTarget struct{}

var _ Target = opencodeTarget{}

// Name is the layout key for the opencode layout.
func (opencodeTarget) Name() string { return "opencode" }

// Present reports whether base holds anything Install could have written for
// this layout: the .opencode dir (skills, the plugin) or the opencode.json
// Install wires at the project root even without .opencode existing yet.
func (opencodeTarget) Present(base string) bool {
	if _, err := os.Stat(filepath.Join(base, ".opencode")); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(base, "opencode.json"))
	return err == nil
}

// Install renders the selected skills under .opencode/skills, then writes
// the root AGENTS.md, wires opencode.json (with the gopls MCP when gopls is
// present), and installs the governance plugin unless NoHooks is set.
func (opencodeTarget) Install(ctx Context) (Report, error) {
	var report Report
	dest := filepath.Join(ctx.Base, opencodewire.SkillsSubdir)
	if err := installSkills(ctx, dest, &report); err != nil {
		return report, err
	}
	gopls := opencodewire.WithoutGopls
	if !ctx.NoMCP {
		if mcpwire.GoplsOnPath() {
			if err := generateGoplsSkill(ctx.Templates, dest, &report); err != nil {
				return report, err
			}
			gopls = opencodewire.WithGopls
		} else {
			report.warn(noGoplsWarning)
		}
	}
	if err := writeAgents(ctx, &report); err != nil {
		return report, err
	}
	if err := retireLegacyAgents(ctx.Base, &report); err != nil {
		return report, err
	}
	if err := opencodewire.WireConfig(ctx.Base, gopls); err != nil {
		return report, fmt.Errorf("wire opencode config: %w", err)
	}
	if !ctx.NoHooks {
		installErr := ctx.HookInstallers.Install("opencode", install.Context{
			Dir:       ctx.Base,
			Templates: ctx.Templates,
		})
		if installErr != nil {
			report.warn("opencode plugin: %v", installErr)
		} else {
			report.note(
				"installed opencode governance plugin: %s",
				filepath.Join(ctx.Base, ".opencode", "plugins"),
			)
		}
	}
	report.note("wired opencode: %s", filepath.Join(ctx.Base, "opencode.json"))
	return report, nil
}

// Uninstall removes the rendered skills and the root AGENTS.md (unless
// ctx.KeepSkills), the opencode.json wiring, and the governance plugin
// Install wrote under ctx.Base, reversing Install. AGENTS.md is gated on
// ctx.KeepSkills exactly like the skill docs, and removal is further gated
// on agentswire's ownership marker: a root AGENTS.md is a name a user very
// plausibly owns already, so only a file this tool actually wrote is removed.
func (opencodeTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	if !ctx.KeepSkills {
		dest := filepath.Join(ctx.Base, opencodewire.SkillsSubdir)
		if err := removeSkills(dest, &report); err != nil {
			return report, err
		}
		if err := removeAgents(ctx.Base, &report); err != nil {
			return report, err
		}
	}
	if err := unwireOpencodeConfig(ctx.Base, &report); err != nil {
		return report, err
	}
	// Gated on actual removal, the same discipline unwireClaudeMCP and
	// removeAgents follow: a project that never installed the opencode target
	// (or already had it removed) must not claim it removed a plugin that was
	// never there.
	result, err := ctx.HookUninstallers.Uninstall(
		"opencode",
		install.UninstallContext{Dir: ctx.Base},
	)
	if err != nil {
		return report, fmt.Errorf("uninstall opencode plugin: %w", err)
	}
	if len(result.Removed) > 0 {
		report.note(
			"removed opencode governance plugin: %s",
			filepath.Join(ctx.Base, ".opencode", "plugins"),
		)
	}
	// Once skills, the plugin, and the binary dirs are gone, .opencode itself
	// may be empty; fsx.RemoveEmptyDir is a no-op when anything of the user's
	// still lives there.
	if err = fsx.RemoveEmptyDir(filepath.Join(ctx.Base, ".opencode")); err != nil {
		return report, fmt.Errorf("remove empty opencode dir: %w", err)
	}
	return report, nil
}

// retireLegacyAgents removes the pre-migration <base>/.opencode/AGENTS.md
// left behind after WriteAgents's output moved from .opencode/AGENTS.md to
// the project root, noting the migration when it fires or warning with the
// exact path when a file sits there that RetireLegacyAgents cannot prove is
// its own - install must never delete a file it cannot verify it wrote.
func retireLegacyAgents(base string, report *Report) error {
	removed, warning, err := opencodewire.RetireLegacyAgents(base)
	if err != nil {
		return fmt.Errorf("retire legacy agents: %w", err)
	}
	if removed {
		report.note(
			"retired legacy %s (superseded by %s)",
			filepath.Join(base, ".opencode", "AGENTS.md"),
			filepath.Join(base, "AGENTS.md"),
		)
	} else if warning != "" {
		report.warn("%s", warning)
	}
	return nil
}

// unwireOpencodeConfig removes the standards wiring from opencode.json,
// warning instead of failing when the file exists but does not parse.
func unwireOpencodeConfig(projectDir string, report *Report) error {
	if err := opencodewire.UnwireConfig(projectDir); err != nil {
		if errors.Is(err, fsx.ErrUnparseableJSON) {
			report.warn("%v", err)
			return nil
		}
		return fmt.Errorf("unwire opencode config: %w", err)
	}
	report.note("unwired %s", filepath.Join(projectDir, "opencode.json"))
	return nil
}
