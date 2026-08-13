package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
)

// claudeInstaller installs the Claude Code hook: shell script + binary +
// settings.json wiring + .gitignore.
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

// Uninstall removes the hook script, README, binary, settings wiring, and
// the "/bin/"/"/hooks/" gitignore entries Install wrote. Hook files and
// gitignore go first, so an unparseable settings file still leaves the
// rest reversed; its error propagates last for the caller to warn on.
// Result.Removed names "gitignore"/"settings" only when that piece changed.
func (claudeInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	restored, err := hookwire.UninstallHook(claudeDir)
	if err != nil {
		return Result{}, fmt.Errorf("uninstall claude hook: %w", err)
	}
	removedEntries, err := gitignore.Remove(claudeDir, "/bin/", "/hooks/")
	if err != nil {
		return Result{}, fmt.Errorf("remove claude gitignore entries: %w", err)
	}
	changed, err := hookwire.UnwireSettings(claudeDir, settingsFile)
	if err != nil {
		return Result{}, fmt.Errorf("unwire claude settings: %w", err)
	}
	result := Result{Restored: restored, SettingsChanged: changed}
	if len(removedEntries) > 0 {
		result.Removed = append(result.Removed, "gitignore")
	}
	if changed {
		result.Removed = append(result.Removed, "settings")
	}
	return result, nil
}
