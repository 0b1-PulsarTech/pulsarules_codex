package vcs

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Change is one path the working tree or the index differs on.
type Change struct {
	// Path is the current, repo-relative path.
	Path string
	// OldPath is the previous path when git reports a staged rename; empty
	// otherwise.
	OldPath string
	// Extension is the lowercase file extension, with the dot.
	Extension string
	// IsTest is true for Go test files (ending in _test.go).
	IsTest bool
	// Staging is git's index-column status code from `git status
	// --porcelain` (e.g. 'M', 'A', 'D', 'R', ' ' when unchanged).
	Staging byte
	// Worktree is git's worktree-column status code from `git status
	// --porcelain` (e.g. 'M', 'D', '?' for untracked, ' ' when unchanged).
	Worktree byte
	// Staged is true when the change is present in the index.
	Staged bool
}

// Status is the set of paths that differ from HEAD.
type Status struct {
	// Changes is the set of changed paths, in the order git reports them.
	Changes []Change
}

// IsClean reports whether no path differs from HEAD.
func (s Status) IsClean() bool {
	return len(s.Changes) == 0
}

// Extensions returns the set of lowercase file extensions present among the
// changed paths.
func (s Status) Extensions() map[string]bool {
	exts := make(map[string]bool, len(s.Changes))
	for _, c := range s.Changes {
		exts[c.Extension] = true
	}
	return exts
}

// String renders the status as porcelain "XY path" lines, in the order git
// reports them, faithfully reproducing git's own status codes.
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

// WorktreeStatus reports the paths that differ from HEAD.
func (r *repository) WorktreeStatus() (Status, error) {
	out, err := runGit(r.root, "status", "--porcelain")
	if err != nil {
		return Status{}, fmt.Errorf("read worktree status: %w", err)
	}
	return parseStatus(out), nil
}

func parseStatus(out string) Status {
	if out == "" {
		return Status{}
	}

	lines := strings.Split(out, "\n")
	changes := make([]Change, 0, len(lines))
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		changes = append(changes, parseStatusLine(line))
	}
	return Status{Changes: changes}
}

// why: porcelain-v1 renders a staged rename as "XY old -> new" instead of
// the plain "XY path" form, so the split-on-arrow branch below is required.
func parseStatusLine(line string) Change {
	staging, worktree := line[0], line[1]
	path, oldPath := line[3:], ""
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
