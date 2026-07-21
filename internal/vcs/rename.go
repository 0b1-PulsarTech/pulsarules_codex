package vcs

import (
	"fmt"
	"strconv"
	"strings"
)

type Rename struct {
	OldPath string
	NewPath string
	Score   int
}

func (repo *repository) StagedRenames(minScore int) ([]Rename, error) {
	out, err := runGit(
		repo.root, "diff", "--cached", "--name-status",
		fmt.Sprintf("--find-renames=%d%%", minScore),
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

func (repo *repository) StagedDiff(path string) (string, error) {
	out, err := runGit(repo.root, "diff", "--cached", "--", path)
	if err != nil {
		return "", fmt.Errorf("read staged diff for %s: %w", path, err)
	}
	return out, nil
}

func (repo *repository) StagedRenameDiff(oldPath, newPath string, minScore int) (string, error) {
	out, err := runGit(
		repo.root, "diff", "--cached", fmt.Sprintf("--find-renames=%d%%", minScore),
		"--", oldPath, newPath,
	)
	if err != nil {
		return "", fmt.Errorf("read staged rename diff for %s -> %s: %w", oldPath, newPath, err)
	}
	return out, nil
}

func parseRenameLine(line string) (Rename, bool) {
	const renameLineFieldCount = 3
	fields := strings.Split(line, "\t")
	if len(fields) != renameLineFieldCount || !strings.HasPrefix(fields[0], "R") {
		return Rename{}, false
	}
	score, err := strconv.Atoi(fields[0][1:])
	if err != nil {
		return Rename{}, false
	}
	return Rename{OldPath: fields[1], NewPath: fields[2], Score: score}, true
}
