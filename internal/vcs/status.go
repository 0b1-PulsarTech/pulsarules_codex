package vcs

import (
	"fmt"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
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

func (s Status) IsClean() bool {
	return len(s.Changes) == 0
}

func (s Status) Extensions() map[string]bool {
	exts := make(map[string]bool, len(s.Changes))
	for _, c := range s.Changes {
		exts[c.Extension] = true
	}
	return exts
}

func (s Status) String() string {
	lines := make([]string, 0, len(s.Changes))
	for _, c := range s.Changes {
		xy := string([]byte{c.Staging, c.Worktree})
		if c.OldPath != "" {
			lines = append(
				lines,
				fmt.Sprintf(
					"%s %s -> %s", xy, printable.Path(c.OldPath), printable.Path(c.Path),
				),
			)
			continue
		}
		lines = append(lines, xy+" "+printable.Path(c.Path))
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
