package install

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

func Opencode(ctx Context) (report.Report, error) {
	rpt, err := installOpencode(ctx)
	if err != nil {
		return rpt, fmt.Errorf("install opencode: %w", err)
	}
	return rpt, nil
}

func UninstallOpencode(ctx UninstallContext) (report.Report, error) {
	rpt, err := uninstallOpencode(ctx)
	if err != nil {
		return rpt, fmt.Errorf("uninstall opencode: %w", err)
	}
	return rpt, nil
}

func installOpencode(ctx Context) (report.Report, error) {
	ledger, err := opencode.Install(ctx.Dir, ctx.Templates)
	if err != nil {
		return report.Report{}, fmt.Errorf("install opencode hook: %w", err)
	}
	rpt := ledger.Report()
	rpt.Note(
		"installed opencode governance plugin: %s",
		filepath.Join(ctx.Dir, ".opencode", "plugins"),
	)
	return rpt, nil
}

func uninstallOpencode(ctx UninstallContext) (report.Report, error) {
	res, err := opencode.Uninstall(ctx.Dir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	// A restore is worth reporting even when nothing else was removed.
	rpt := res.Report()
	if res.Removed {
		rpt.Note(
			"removed opencode governance plugin: %s",
			filepath.Join(ctx.Dir, ".opencode", "plugins"),
		)
	}
	return rpt, nil
}
