package install

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

// gitInstaller installs git hook scripts into .git/hooks/.
type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

// Install writes exactly ctx.GitHooks, verbatim - including nothing for a
// caller-supplied empty list. The "commit-msg,pre-commit" default lives
// solely on the --git-hooks flag, so the printed line and hooks actually
// written always agree; a second default here would let an explicit empty
// list silently resurrect it.
func (gitInstaller) Install(ctx Context) (report.Report, error) {
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

// Uninstall removes the git hook scripts and installer binary Install wrote
// into ctx.Dir/.git/hooks/. The "removed git hooks" note fires only once
// nothing failed, mirroring githook.Uninstall's all-or-partial return; the
// restore notes print regardless, since a restore that already happened is
// worth reporting even if a later step in the same call errors.
func (gitInstaller) Uninstall(ctx UninstallContext) (report.Report, error) {
	removed, restored, err := githook.Uninstall(ctx.Dir)
	var rpt report.Report
	if err == nil && len(removed) > 0 {
		rpt.Note("removed git hooks")
	}
	for _, msg := range restored {
		rpt.Note("%s", msg)
	}
	if err != nil {
		return rpt, fmt.Errorf("uninstall git hooks: %w", err)
	}
	return rpt, nil
}
