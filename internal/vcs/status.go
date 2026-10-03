package vcs

import (
	"fmt"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

type Change struct {
	Path      string
	OldPath   string
	Extension string
	IsTest    bool
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

func (status Status) String() string {
	lines := make([]string, 0, len(status.Changes))
	for _, change := range status.Changes {
		xy := string([]byte{change.Staging, change.Worktree})
		if change.OldPath != "" {
			lines = append(
				lines,
				fmt.Sprintf(
					"%s %s -> %s", xy, escape.Path(change.OldPath), escape.Path(change.Path),
				),
			)
			continue
		}
		lines = append(lines, xy+" "+escape.Path(change.Path))
	}
	return strings.Join(lines, "\n")
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
