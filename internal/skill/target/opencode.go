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

type opencodeTarget struct{}

var _ Target = opencodeTarget{}

func (opencodeTarget) Name() string { return "opencode" }

func (opencodeTarget) Present(base string) bool {
	if _, err := os.Stat(filepath.Join(base, ".opencode")); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(base, "opencode.json"))
	return err == nil
}

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
			report.Warn(noGoplsWarning)
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
		sub, installErr := ctx.HookInstallers.Install("opencode", install.Context{
			Dir:       ctx.Base,
			Templates: ctx.Templates,
		})
		report.Merge(sub)
		if installErr != nil {
			report.Warn("opencode plugin: %v", installErr)
		}
	}
	report.Note("wired opencode: %s", filepath.Join(ctx.Base, "opencode.json"))
	return report, nil
}

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
	sub, err := ctx.HookUninstallers.Uninstall(
		"opencode",
		install.UninstallContext{Dir: ctx.Base},
	)
	if err != nil {
		return report, fmt.Errorf("uninstall opencode plugin: %w", err)
	}
	report.Merge(sub)
	if err = fsx.RemoveEmptyDir(filepath.Join(ctx.Base, ".opencode")); err != nil {
		return report, fmt.Errorf("remove empty opencode dir: %w", err)
	}
	return report, nil
}

func retireLegacyAgents(base string, report *Report) error {
	removed, warning, err := opencodewire.RetireLegacyAgents(base)
	if err != nil {
		return fmt.Errorf("retire legacy agents: %w", err)
	}
	if removed {
		report.Note(
			"retired legacy %s (superseded by %s)",
			filepath.Join(base, ".opencode", "AGENTS.md"),
			filepath.Join(base, "AGENTS.md"),
		)
	} else if warning != "" {
		report.Warn("%s", warning)
	}
	return nil
}

func unwireOpencodeConfig(projectDir string, report *Report) error {
	changed, err := opencodewire.UnwireConfig(projectDir)
	if err != nil {
		if errors.Is(err, fsx.ErrUnparseableJSON) {
			report.Warn("%v", err)
			return nil
		}
		return fmt.Errorf("unwire opencode config: %w", err)
	}
	if changed {
		report.Note("unwired %s", filepath.Join(projectDir, "opencode.json"))
	}
	return nil
}
