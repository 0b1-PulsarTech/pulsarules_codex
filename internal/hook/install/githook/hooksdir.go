package githook

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func hooksDir(dir string) string {
	// A linked worktree's .git is a file, so ask git for the common dir.
	commonDir, err := vcs.CommonDir(dir)
	if err != nil {
		// A tree that only looks like a repository still installs under dir/.git.
		return filepath.Join(dir, ".git", "hooks")
	}
	return filepath.Join(commonDir, "hooks")
}

func Orphans(dir string) (notes []string, err error) {
	dest := hooksDir(dir)
	for _, name := range HookNames() {
		path := filepath.Join(dest, name)
		slots, slotsErr := marker.Orphans(path)
		if slotsErr != nil {
			return notes, fmt.Errorf("list backups of %q: %w", path, slotsErr)
		}
		if len(slots) > 0 {
			notes = append(notes, marker.OrphanMessage(path, slots))
		}
	}
	return notes, nil
}
