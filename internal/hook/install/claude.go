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

// Uninstall removes the hook script, README, binary, and settings wiring
// Install wrote into ctx.Dir, plus the "/bin/" and "/hooks/" gitignore
// entries. The hook files and gitignore entries are removed first, so a
// settings file UnwireSettings cannot parse still leaves the rest of the
// reversal done; its error (wrapping fsx.ErrUnparseableJSON) propagates last
// for the caller to turn into a warning instead of a hard failure.
// Result.Removed carries "gitignore" and/or "settings" only when that piece
// actually changed, so a caller can tell a real removal from a no-op against
// files the hook was never wired into.
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
	result := Result{Restored: restored}
	if len(removedEntries) > 0 {
		result.Removed = append(result.Removed, "gitignore")
	}
	if changed {
		result.Removed = append(result.Removed, "settings")
	}
	return result, nil
}
