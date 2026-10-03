package install

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(ctx Context) (report.Report, error) {
	// Pass ctx.GitHooks verbatim: the default lives only on --git-hooks.
	backedUp, err := githook.Install(ctx.Dir, ctx.GitHooks, githook.Options{
		TypographicSeverity: ctx.TypographicSeverity,
		BranchExtraTypes:    ctx.BranchExtraTypes,
	})
	if err != nil {
		return report.Report{}, fmt.Errorf("install git hooks: %w", err)
	}
	var rpt report.Report
	for _, msg := range backedUp {
		rpt.Warn("%s", msg)
	}
	if len(ctx.GitHooks) > 0 {
		rpt.Note("installed git hooks: %s", strings.Join(ctx.GitHooks, ","))
	}
	return rpt, nil
}

func (gitInstaller) Uninstall(ctx UninstallContext) (report.Report, error) {
	res, err := githook.Uninstall(ctx.Dir)
	var rpt report.Report
	if err == nil && len(res.Removed) > 0 {
		rpt.Note("removed git hooks")
	}
	// Orphan and restore notes print even on error: completed work is worth reporting.
	for _, msg := range res.Orphaned {
		rpt.Warn("%s", msg)
	}
	for _, msg := range res.Restored {
		rpt.Note("%s", msg)
	}
	if err != nil {
		return rpt, fmt.Errorf("uninstall git hooks: %w", err)
	}
	return rpt, nil
}
