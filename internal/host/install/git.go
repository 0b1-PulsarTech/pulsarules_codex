package install

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func Git(ctx Context) (report.Report, error) {
	rpt, err := installGit(ctx)
	if err != nil {
		return rpt, fmt.Errorf("install git: %w", err)
	}
	return rpt, nil
}

func UninstallGit(ctx UninstallContext) (report.Report, error) {
	rpt, err := uninstallGit(ctx)
	if err != nil {
		return rpt, fmt.Errorf("uninstall git: %w", err)
	}
	return rpt, nil
}

func installGit(ctx Context) (report.Report, error) {
	// Pass ctx.GitHooks verbatim: the default lives only on --git-hooks.
	ledger, err := githook.Install(ctx.Dir, ctx.GitHooks, githook.Options{
		TypographicSeverity: ctx.TypographicSeverity,
		BranchExtraTypes:    ctx.BranchExtraTypes,
	})
	if err != nil {
		return report.Report{}, fmt.Errorf("install git hooks: %w", err)
	}
	rpt := ledger.Report()
	if len(ctx.GitHooks) > 0 {
		rpt.Note("installed git hooks: %s", strings.Join(ctx.GitHooks, ","))
	}
	return rpt, nil
}

func uninstallGit(ctx UninstallContext) (report.Report, error) {
	res, err := githook.Uninstall(ctx.Dir)
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
