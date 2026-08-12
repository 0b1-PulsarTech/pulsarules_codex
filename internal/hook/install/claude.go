package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
)

type claudeInstaller struct{}

func (claudeInstaller) Name() string { return "claude" }

func (claudeInstaller) Install(ctx Context) error {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	backedUp, err := hookwire.InstallHook(ctx.Templates, claudeDir)
	if err != nil {
		return fmt.Errorf("install claude hook: %w", err)
	}
	if ctx.Warn != nil {
		for _, msg := range backedUp {
			ctx.Warn("%s", msg)
		}
	}
	if err = hookwire.WireSettings(ctx.Templates, claudeDir, settingsFile); err != nil {
		return fmt.Errorf("wire claude settings: %w", err)
	}
	if err = gitignore.Ensure(claudeDir, "/bin/", "/hooks/"); err != nil {
		return fmt.Errorf("ensure claude gitignore: %w", err)
	}
	return nil
}
