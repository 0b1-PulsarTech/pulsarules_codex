package install

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

func Opencode(dir string, templates fs.FS) (report.Report, error) {
	rpt, err := installOpencode(dir, templates)
	if err != nil {
		return rpt, fmt.Errorf("install opencode: %w", err)
	}
	return rpt, nil
}

func UninstallOpencode(dir string) (report.Report, error) {
	rpt, err := uninstallOpencode(dir)
	if err != nil {
		return rpt, fmt.Errorf("uninstall opencode: %w", err)
	}
	return rpt, nil
}

func installOpencode(dir string, templates fs.FS) (report.Report, error) {
	ledger, err := opencode.InstallPlugin(dir, templates)
	if err != nil {
		return report.Report{}, fmt.Errorf("install opencode hook: %w", err)
	}
	rpt := ledger.Report()
	rpt.Note(
		"installed opencode governance plugin: %s",
		filepath.Join(dir, ".opencode", "plugins"),
	)
	return rpt, nil
}

func uninstallOpencode(dir string) (report.Report, error) {
	res, err := opencode.UninstallPlugin(dir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	// A restore is worth reporting even when nothing else was removed.
	rpt := res.Report()
	if res.Removed {
		rpt.Note(
			"removed opencode governance plugin: %s",
			filepath.Join(dir, ".opencode", "plugins"),
		)
	}
	return rpt, nil
}
