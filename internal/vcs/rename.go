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
	// -z keeps non-ASCII paths unquoted; the --no-* flags keep user diff config out of the parse.
	out, err := runGit(
		repo.root, "diff", "--cached", "--name-status", "-z",
		"--no-color", "--no-ext-diff", "--no-textconv",
		fmt.Sprintf("--find-renames=%d%%", minScore),
	)
	if err != nil {
		return nil, fmt.Errorf("read staged renames: %w", err)
	}
	return parseNameStatus(strings.Split(out, "\x00")), nil
}

func parseNameStatus(fields []string) []Rename {
	const (
		nameStatusSingleWidth = 2
		nameStatusPairWidth   = 3
	)
	var renames []Rename
	// A rename or copy line is status, old, new; any other is status, path.
	for pos := 0; pos < len(fields); {
		status := fields[pos]
		if status == "" {
			pos++
			continue
		}
		isPair := strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C")
		width := nameStatusSingleWidth
		if isPair {
			width = nameStatusPairWidth
		}
		if pos+width > len(fields) {
			break
		}
		if score, err := strconv.Atoi(status[1:]); err == nil && strings.HasPrefix(status, "R") {
			renames = append(renames, Rename{
				OldPath: fields[pos+1], NewPath: fields[pos+2], Score: score,
			})
		}
		pos += width
	}
	return renames
}
