package vcs

import (
	"fmt"
	"path/filepath"
)

// CommonDir returns the git directory every linked worktree of dir's repository
// shares - the one holding the hooks git actually runs.
// why: in a linked worktree dir/.git is a FILE pointing at the real gitdir, so
// joining dir with ".git" yields ENOTDIR and an installer writing there reaches
// nothing. git reports it relative in a checkout, absolute in a worktree.
func CommonDir(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("read git common dir: %w", err)
	}
	if out == "" {
		return "", fmt.Errorf("read git common dir: %w: empty path", ErrNoRepository)
	}
	if filepath.IsAbs(out) {
		return out, nil
	}
	return filepath.Join(dir, out), nil
}
