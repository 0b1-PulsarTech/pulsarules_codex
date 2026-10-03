package vcs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (repo *repository) ListFiles() ([]string, error) {
	const listTimeout = 30 * time.Second
	// --exclude-standard also applies the user's global excludesFile and .git/info/exclude.
	// A whole-tree listing on a large repo outlasts the 2s default.
	out, err := runGitWithin(listTimeout, repo.root,
		"ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	var files []string
	for entry := range strings.SplitSeq(out, "\x00") {
		if entry == "" {
			continue
		}
		// ls-files still names a cached file deleted from disk and a
		// submodule gitlink (a directory); neither is a readable file.
		stat, statErr := os.Lstat(filepath.Join(repo.root, filepath.FromSlash(entry)))
		if statErr != nil || stat.IsDir() {
			continue
		}
		files = append(files, entry)
	}
	slices.Sort(files)
	// An unmerged path is listed once per index stage.
	return slices.Compact(files), nil
}
