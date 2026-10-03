package vcs

import (
	"path/filepath"
	"strings"
)

const porcelainPrefixLength = 3

func parseStatus(out string) Status {
	fields := strings.Split(out, "\x00")
	changes := make([]Change, 0, len(fields))
	for pos := 0; pos < len(fields); pos++ {
		entry := fields[pos]
		if len(entry) < porcelainPrefixLength {
			continue
		}
		oldPath := ""
		// -z renders a rename or copy as "XY new" then the old path as its own field.
		if isPairEntry(entry) && pos+1 < len(fields) {
			pos++
			oldPath = fields[pos]
		}
		changes = append(changes, newChange(entry, oldPath))
	}
	return Status{Changes: changes}
}

func isPairEntry(entry string) bool {
	return strings.ContainsAny(entry[:2], "RC")
}

func newChange(entry, oldPath string) Change {
	staging, worktree := entry[0], entry[1]
	// Two status bytes and a separating space.
	path := entry[porcelainPrefixLength:]
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
