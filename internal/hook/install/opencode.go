package install

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

// opencodeInstaller installs the opencode governance plugin + binary +
// .gitignore.
type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

// Install writes the governance plugin, binary, and gitignore entry into
// ctx.Dir, noting the install once every step succeeds. A foreign file
// Install backs up rather than overwrites is discarded from the Report on a
// later failure, matching opencodehook.Install's own partial-result
// discipline.
func (opencodeInstaller) Install(ctx Context) (report.Report, error) {
	backedUp, err := opencodehook.Install(ctx.Dir, ctx.Templates)
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

// Uninstall removes the governance plugin, binary, and gitignore entry
// Install wrote into ctx.Dir. The note fires only when a plugin actually
// existed, so a caller can tell a real removal from a no-op against a
// project Install never touched.
func (opencodeInstaller) Uninstall(ctx UninstallContext) (report.Report, error) {
	removed, restored, err := opencodehook.Uninstall(ctx.Dir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	var rpt report.Report
	// why: a restore that happened is worth reporting even when nothing else
	// was removed - the foreign file it uncovers is back on disk either way.
	for _, msg := range restored {
		rpt.Note("%s", msg)
	}
	if removed {
		rpt.Note(
			"removed opencode governance plugin: %s",
			filepath.Join(ctx.Dir, ".opencode", "plugins"),
		)
	}
	return rpt, nil
}
