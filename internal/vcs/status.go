package vcs

import (
	"fmt"
	"time"
)

type Change struct {
	Path      string
	OldPath   string
	Extension string
	Staging   byte
	Worktree  byte
	Staged    bool
}

type Status struct {
	Changes []Change
}

func (status Status) IsClean() bool {
	return len(status.Changes) == 0
}

func (status Status) Extensions() map[string]bool {
	exts := make(map[string]bool, len(status.Changes))
	for _, change := range status.Changes {
		exts[change.Extension] = true
	}
	return exts
}

func (repo *repository) WorktreeStatus() (Status, error) {
	const statusTimeout = 30 * time.Second
	// --no-optional-locks keeps a background run from rewriting the user's index.
	// A whole-tree untracked listing on a large repo outlasts the 2s default.
	out, err := runGitWithin(statusTimeout, repo.root, "--no-optional-locks",
		"status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return Status{}, fmt.Errorf("read worktree status: %w", err)
	}
	return parseStatus(out), nil
}
