package install

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

func Claude(ctx Context) (report.Report, error) {
	rpt, err := installClaude(ctx)
	if err != nil {
		return rpt, fmt.Errorf("install claude: %w", err)
	}
	return rpt, nil
}

func UninstallClaude(ctx UninstallContext) (report.Report, error) {
	rpt, err := uninstallClaude(ctx)
	if err != nil {
		return rpt, fmt.Errorf("uninstall claude: %w", err)
	}
	return rpt, nil
}

func installClaude(ctx Context) (report.Report, error) {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	ledger, err := claude.InstallHook(ctx.Templates, claudeDir)
	if err != nil {
		return report.Report{}, fmt.Errorf("install claude hook: %w", err)
	}
	rpt := ledger.Report()
	if err = claude.WireSettings(claudeDir, settingsFile); err != nil {
		return rpt, fmt.Errorf("wire claude settings: %w", err)
	}
	if err = gitignore.Ensure(claudeDir, "/bin/", "/hooks/"); err != nil {
		return rpt, fmt.Errorf("ensure claude gitignore: %w", err)
	}
	rpt.Note("wired hook into %s", filepath.Join(claudeDir, settingsFile))
	return rpt, nil
}

func uninstallClaude(ctx UninstallContext) (report.Report, error) {
	claudeDir := ctx.Dir
	settingsFile := ctx.SettingsFile
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	ledger, err := claude.UninstallHook(claudeDir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall claude hook: %w", err)
	}
	if _, err = gitignore.Remove(claudeDir, "/bin/", "/hooks/"); err != nil {
		return report.Report{}, fmt.Errorf("remove claude gitignore entries: %w", err)
	}
	// Settings go last: an unparseable file still leaves the rest reversed.
	changed, err := claude.UnwireSettings(claudeDir, settingsFile)
	if err != nil {
		return report.Report{}, fmt.Errorf("unwire claude settings: %w", err)
	}
	var rpt report.Report
	if changed {
		rpt.Note("removed hook wiring from %s", filepath.Join(claudeDir, settingsFile))
	}
	rpt.Merge(ledger.Report())
	return rpt, nil
}
