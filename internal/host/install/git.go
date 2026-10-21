package install

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func Git(dir string, gitHooks []string, options githook.Options) (report.Report, error) {
	rpt, err := installGit(dir, gitHooks, options)
	if err != nil {
		return rpt, fmt.Errorf("install git: %w", err)
	}
	return rpt, nil
}

func UninstallGit(dir string) (report.Report, error) {
	rpt, err := uninstallGit(dir)
	if err != nil {
		return rpt, fmt.Errorf("uninstall git: %w", err)
	}
	return rpt, nil
}

func installGit(
	dir string, gitHooks []string, options githook.Options,
) (report.Report, error) {
	// Pass gitHooks verbatim: the default lives only on --git-hooks.
	ledger, err := githook.Install(dir, gitHooks, options)
	if err != nil {
		return report.Report{}, fmt.Errorf("install git hooks: %w", err)
	}
	rpt := ledger.Report()
	if len(gitHooks) > 0 {
		rpt.Note("installed git hooks: %s", strings.Join(gitHooks, ","))
	}
	return rpt, nil
}

func uninstallGit(dir string) (report.Report, error) {
	res, err := githook.Uninstall(dir)
	var rpt report.Report
	if err == nil && len(res.Removed) > 0 {
		rpt.Note("removed git hooks")
	}
	// Orphan and restore notes print even on error: completed work is worth reporting.
	rpt.Merge(res.Report())
	if err != nil {
		return rpt, fmt.Errorf("uninstall git hooks: %w", err)
	}
	return rpt, nil
}
