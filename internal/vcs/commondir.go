package vcs

import (
	"fmt"
	"path/filepath"
)

func CommonDir(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("read git common dir: %w", err)
	}
	if out == "" {
		return "", fmt.Errorf("read git common dir: %w: empty path", ErrNoRepository)
	}
	// In a linked worktree dir/.git is a file, and git prints the path relative or absolute.
	if filepath.IsAbs(out) {
		return out, nil
	}
	return filepath.Join(dir, out), nil
}
