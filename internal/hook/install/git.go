package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
)

// gitInstaller installs git hook scripts into .git/hooks/.
type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

// Install writes exactly ctx.GitHooks, verbatim - including nothing for a
// caller-supplied empty list. The "commit-msg,pre-commit" default lives
// solely on the --git-hooks flag, so the printed line and hooks actually
// written always agree; a second default here would let an explicit empty
// list silently resurrect it.
func (gitInstaller) Install(ctx Context) error {
	backedUp, err := githook.Install(ctx.Dir, ctx.GitHooks, githook.Options{
		TypographicSeverity: ctx.TypographicSeverity,
	})
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

// Uninstall removes the git hook scripts and installer binary Install wrote into
// the repository's shared hooks dir, reporting which hook names were removed,
// which backups were restored, and which earlier backup slots were left behind.
func (gitInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	removed, restored, err := githook.Uninstall(ctx.Dir)
	result := Result{Removed: removed, Restored: restored}
	if err != nil {
		return result, fmt.Errorf("uninstall git hooks: %w", err)
	}
	// Queried after the removal: Restore consumes only the base backup slot, so
	// any earlier one is still there and nothing else would ever mention it.
	notes, notesErr := githook.Orphans(ctx.Dir)
	if notesErr != nil {
		return result, fmt.Errorf("list leftover git hook backups: %w", notesErr)
	}
	result.Notes = notes
	return result, nil
}
