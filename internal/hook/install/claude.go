package install

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
)

// claudeInstaller installs the Claude Code hook: shell script + binary +
// settings.json wiring + .gitignore.
type claudeInstaller struct{}

func (claudeInstaller) Name() string { return "claude" }

// Install writes the hook script, binary, settings wiring, and gitignore
// entries into ctx.Dir, noting the wire once every step succeeds. A foreign
// file InstallHook backs up rather than overwrites is discarded from the
// Report on a later failure, matching InstallHook's own partial-result
// discipline: a caller only learns of a backup once the whole install lands.
func (claudeInstaller) Install(ctx Context) (report.Report, error) {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	backedUp, err := hookwire.InstallHook(ctx.Templates, claudeDir)
	if err != nil {
		return report.Report{}, fmt.Errorf("install claude hook: %w", err)
	}
	var rpt report.Report
	for _, msg := range backedUp {
		rpt.Warn("%s", msg)
	}
	if err = hookwire.WireSettings(ctx.Templates, claudeDir, settingsFile); err != nil {
		return rpt, fmt.Errorf("wire claude settings: %w", err)
	}
	if err = gitignore.Ensure(claudeDir, "/bin/", "/hooks/"); err != nil {
		return rpt, fmt.Errorf("ensure claude gitignore: %w", err)
	}
	rpt.Note("wired hook into %s", filepath.Join(claudeDir, settingsFile))
	return rpt, nil
}

// Uninstall removes the hook script, README, binary, settings wiring, and
// the "/bin/"/"/hooks/" gitignore entries Install wrote. Hook files and
// gitignore go first, so an unparseable settings file still leaves the rest
// reversed. An error discards only the caller-visible Report - the disk
// work already done (removals, restores) stands, not rolled back.
func (claudeInstaller) Uninstall(ctx UninstallContext) (report.Report, error) {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	restored, orphaned, err := hookwire.UninstallHook(claudeDir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall claude hook: %w", err)
	}
	if _, err = gitignore.Remove(claudeDir, "/bin/", "/hooks/"); err != nil {
		return report.Report{}, fmt.Errorf("remove claude gitignore entries: %w", err)
	}
	changed, err := hookwire.UnwireSettings(claudeDir, settingsFile)
	if err != nil {
		return report.Report{}, fmt.Errorf("unwire claude settings: %w", err)
	}
	var rpt report.Report
	if changed {
		rpt.Note("removed hook wiring from %s", filepath.Join(claudeDir, settingsFile))
	}
	for _, msg := range orphaned {
		rpt.Warn("%s", msg)
	}
	for _, msg := range restored {
		rpt.Note("%s", msg)
	}
	return rpt, nil
}
