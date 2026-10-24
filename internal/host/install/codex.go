package install

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/codex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

func codexIgnored() []string { return []string{"/bin/", "/" + codex.LogFileName} }

func Codex(codexDir string, templates fs.FS, loc codex.Location) (report.Report, error) {
	ledger, err := codex.InstallHook(templates, codexDir, loc)
	if err != nil {
		return report.Report{}, fmt.Errorf("install codex hook: %w", err)
	}
	rpt := ledger.Report()
	wired, err := wireHooks(codexDir, loc, &rpt)
	if err != nil {
		return rpt, err
	}
	if loc.GlobalDir == "" {
		// A user-level directory is no repository; the project script stays committed so a
		// fresh clone is quiet until the binary exists.
		if err = gitignore.Ensure(codexDir, codexIgnored()...); err != nil {
			return rpt, fmt.Errorf("ensure codex gitignore: %w", err)
		}
	}
	if wired {
		rpt.Note("wired hooks into %s", filepath.Join(codexDir, codex.HooksFile))
	}
	return rpt, nil
}

func wireHooks(codexDir string, loc codex.Location, rpt *report.Report) (wired bool, err error) {
	err = codex.WireHooks(codexDir, loc)
	if !errors.Is(err, codex.ErrSymlink) {
		if err != nil {
			return false, fmt.Errorf("wire codex hooks: %w", err)
		}
		return true, nil
	}
	// The report prints one escaped line per entry, so the block comes from --print-hooks.
	scope := "--project " + filepath.Dir(codexDir)
	if loc.GlobalDir != "" {
		scope = "--global"
	}
	rpt.Warn("%v; left it alone. Merge the output of `pulsarules_cli install --print-hooks "+
		"--target codex %s` into the file it points to so Codex runs the hooks", err, scope)
	return false, nil
}

func UninstallCodex(codexDir string) (report.Report, error) {
	ledger, err := codex.UninstallHook(codexDir)
	if err != nil {
		return report.Report{}, fmt.Errorf("uninstall codex hook: %w", err)
	}
	if _, err = gitignore.Remove(codexDir, codexIgnored()...); err != nil {
		return report.Report{}, fmt.Errorf("remove codex gitignore entries: %w", err)
	}
	// The hooks file goes last: an unparseable one still leaves the rest reversed.
	changed, err := codex.UnwireHooks(codexDir)
	var rpt report.Report
	switch {
	case errors.Is(err, codex.ErrSymlink):
		rpt.Warn("%v; left it alone. Remove our hook entries from it yourself", err)
	case err != nil:
		return report.Report{}, fmt.Errorf("unwire codex hooks: %w", err)
	case changed:
		rpt.Note("removed hook wiring from %s", filepath.Join(codexDir, codex.HooksFile))
	}
	rpt.Merge(ledger.Report())
	return rpt, nil
}
