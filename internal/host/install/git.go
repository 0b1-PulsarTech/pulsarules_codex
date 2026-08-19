package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(ctx Context) error {
	// Pass ctx.GitHooks verbatim: the default lives only on --git-hooks.
	backedUp, err := githook.Install(ctx.Dir, ctx.GitHooks)
	if err != nil {
		return fmt.Errorf("install git hooks: %w", err)
	}
	if ctx.Warn != nil {
		for _, msg := range backedUp {
			ctx.Warn("%s", msg)
		}
	}
	return nil
}

func (gitInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	removed, restored, err := githook.Uninstall(ctx.Dir)
	result := Result{Removed: removed, Restored: restored}
	if err != nil {
		return result, fmt.Errorf("uninstall git hooks: %w", err)
	}
	notes, notesErr := githook.Orphans(ctx.Dir)
	if notesErr != nil {
		return result, fmt.Errorf("list leftover git hook backups: %w", notesErr)
	}
	result.Notes = notes
	return result, nil
}
