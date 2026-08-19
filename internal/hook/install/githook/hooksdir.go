package githook

import (
	"path/filepath"

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
