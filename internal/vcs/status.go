package vcs

import (
	"fmt"
	"path/filepath"
	"strings"
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
			lines = append(lines, fmt.Sprintf("%s %s -> %s", xy, c.OldPath, c.Path))
			continue
		}
		lines = append(lines, xy+" "+c.Path)
	}
	return strings.Join(lines, "\n")
}

func (repo *repository) WorktreeStatus() (Status, error) {
	out, err := runGit(repo.root, "status", "--porcelain")
	if err != nil {
		return Status{}, fmt.Errorf("read worktree status: %w", err)
	}
	return parseStatus(out), nil
}

const porcelainPrefixLength = 3

func parseStatus(out string) Status {
	if out == "" {
		return Status{}
	}

	lines := strings.Split(out, "\n")
	changes := make([]Change, 0, len(lines))
	for _, line := range lines {
		if len(line) < porcelainPrefixLength {
			continue
		}
		changes = append(changes, parseStatusLine(line))
	}
	return Status{Changes: changes}
}

func parseStatusLine(line string) Change {
	staging, worktree := line[0], line[1]
	path, oldPath := line[porcelainPrefixLength:], ""
	if before, after, found := strings.Cut(path, " -> "); found {
		oldPath, path = before, after
	}
	return Change{
		Path:      path,
		OldPath:   oldPath,
		Extension: strings.ToLower(filepath.Ext(path)),
		IsTest:    strings.HasSuffix(path, "_test.go"),
		Staging:   staging,
		Worktree:  worktree,
		Staged:    staging != ' ' && staging != '?',
	}
}
