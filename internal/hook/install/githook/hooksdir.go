package githook

import (
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

// hooksDir resolves the dir git runs hook scripts from. A linked worktree shares
// the main repository's hooks and its own .git is a file, so joining dir with
// ".git" reaches nothing there. The fallback keeps a tree that only LOOKS like a
// repository - no git binary, or a bare .git dir - installing as it did before.
func hooksDir(dir string) string {
	commonDir, err := vcs.CommonDir(dir)
	if err != nil {
		return filepath.Join(dir, ".git", "hooks")
	}
	return filepath.Join(commonDir, "hooks")
}

// Orphans reports the earlier backup slots left beside each hook path, as
// ready-to-print notes. It is a query, deliberately separate from Uninstall's
// command: Restore consumes only the base slot, so a numbered one survives an
// uninstall and would otherwise sit in .git/hooks unmentioned forever.
func Orphans(dir string) (notes []string, err error) {
	dest := hooksDir(dir)
	paths := make([]string, 0, len(HookNames()))
	for _, name := range HookNames() {
		paths = append(paths, filepath.Join(dest, name))
	}
	return marker.OrphanNotes(paths...)
}
