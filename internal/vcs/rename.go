package vcs

import (
	"fmt"
	"strconv"
	"strings"
)

// Rename is a staged delete+add pair git's own rename detection scored as a
// move.
type Rename struct {
	// OldPath is the staged-deleted path.
	OldPath string
	// NewPath is the staged-added path judged to be its replacement.
	NewPath string
	// Score is git's rename-similarity score, from 0 to 100.
	Score int
}

// StagedRenames pairs staged deletions with staged additions that look like
// the same file moved, scored at or above minScore. Git does its own
// content-similarity detection; this only parses the result.
func (r *repository) StagedRenames(minScore int) ([]Rename, error) {
	out, err := runGit(
		r.root, "diff", "--cached", "--name-status", fmt.Sprintf("--find-renames=%d%%", minScore),
	)
	if err != nil {
		return nil, fmt.Errorf("read staged renames: %w", err)
	}
	if out == "" {
		return nil, nil
	}

	var renames []Rename
	for line := range strings.SplitSeq(out, "\n") {
		if rename, ok := parseRenameLine(line); ok {
			renames = append(renames, rename)
		}
	}
	return renames, nil
}

// StagedDiff returns the unified diff body for a single staged path (git
// diff --cached -- path), so a caller can inspect what changed inside a
// file without shelling out itself.
func (r *repository) StagedDiff(path string) (string, error) {
	out, err := runGit(r.root, "diff", "--cached", "--", path)
	if err != nil {
		return "", fmt.Errorf("read staged diff for %s: %w", path, err)
	}
	return out, nil
}

// StagedRenameDiff returns the unified diff body for a staged rename pair
// (oldPath deleted, newPath added), scored at or above minScore, so a
// caller can inspect what changed within the move itself rather than
// seeing newPath's full content as a fresh addition. A pathspec naming only
// newPath hides the paired deletion from git's own rename detection, so
// both paths are required to get the delta rather than the whole file.
func (r *repository) StagedRenameDiff(oldPath, newPath string, minScore int) (string, error) {
	out, err := runGit(
		r.root, "diff", "--cached", fmt.Sprintf("--find-renames=%d%%", minScore),
		"--", oldPath, newPath,
	)
	if err != nil {
		return "", fmt.Errorf("read staged rename diff for %s -> %s: %w", oldPath, newPath, err)
	}
	return out, nil
}

func parseRenameLine(line string) (Rename, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) != 3 || !strings.HasPrefix(fields[0], "R") {
		return Rename{}, false
	}
	score, err := strconv.Atoi(fields[0][1:])
	if err != nil {
		return Rename{}, false
	}
	return Rename{OldPath: fields[1], NewPath: fields[2], Score: score}, true
}
