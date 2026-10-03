package install

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(ctx Context) (report.Report, error) {
	backedUp, err := opencode.Install(ctx.Dir, ctx.Templates)
	if err != nil {
		return report.Report{}, fmt.Errorf("install opencode hook: %w", err)
	}
	var rpt report.Report
	for _, msg := range backedUp {
		rpt.Warn("%s", msg)
	}
	rpt.Note(
		"installed opencode governance plugin: %s",
		filepath.Join(ctx.Dir, ".opencode", "plugins"),
	)
	return rpt, nil
}

func (opencodeInstaller) Uninstall(ctx UninstallContext) (report.Report, error) {
	res, err := opencode.Uninstall(ctx.Dir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	var rpt report.Report
	// A restore is worth reporting even when nothing else was removed.
	for _, msg := range res.Restored {
		rpt.Note("%s", msg)
	}
	for _, msg := range res.Orphaned {
		rpt.Warn("%s", msg)
	}
	if res.Removed {
		rpt.Note(
			"removed opencode governance plugin: %s",
			filepath.Join(ctx.Dir, ".opencode", "plugins"),
		)
	}
	return rpt, nil
}
