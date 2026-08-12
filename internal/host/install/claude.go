package install

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

type claudeInstaller struct{}

func (claudeInstaller) Name() string { return "claude" }

func (claudeInstaller) Install(dir string, templates fs.FS, settingsFile string) error {
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	if err := claude.InstallHook(templates, dir); err != nil {
		return fmt.Errorf("install claude hook: %w", err)
	}
	if err := claude.WireSettings(templates, dir, settingsFile); err != nil {
		return fmt.Errorf("wire claude settings: %w", err)
	}
	if err := gitignore.Ensure(dir, "/bin/", "/hooks/"); err != nil {
		return fmt.Errorf("ensure claude gitignore: %w", err)
	}
	return nil
}
