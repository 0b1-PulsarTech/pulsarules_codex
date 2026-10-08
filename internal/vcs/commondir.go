package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrGitNotFound = errors.New("vcs: git executable not found")

func CommonDir(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", classifyCommonDir(err)
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

func classifyCommonDir(err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("read git common dir: %w: %w", ErrGitNotFound, err)
	}
	// Only git's own "nowhere in a repository" wording; a broken gitdir pointer reads differently.
	if gitErr, ok := errors.AsType[*gitError](err); ok &&
		strings.Contains(gitErr.stderr, "not a git repository (or any") {
		return fmt.Errorf("read git common dir: %w: %w", ErrNoRepository, err)
	}
	return fmt.Errorf("read git common dir: %w", err)
}
