package vcs

import (
	"fmt"
	"path"
	"strings"
)

func (repo *Repository) Entry(side Side, path string) (mode, id string, err error) {
	args := []string{"--literal-pathspecs", "ls-tree", "-z", "HEAD", "--", path}
	idField := 2
	if side == Staged {
		args = []string{"--literal-pathspecs", "ls-files", "-s", "-z", "--", path}
		idField = 1
	}
	var out string
	out, err = runGit(repo.root, args...)
	if err != nil {
		return "", "", fmt.Errorf("read %s entry %s: %w", side, path, err)
	}
	mode, id, err = parseEntry(out, path, idField)
	if err != nil {
		return "", "", fmt.Errorf("read %s entry %s: %w", side, path, err)
	}
	return mode, id, nil
}

func (repo *Repository) Dir(side Side, dir string) ([]string, error) {
	args := []string{"--literal-pathspecs", "ls-tree", "-z", "--name-only", "HEAD"}
	if side == Staged {
		args = []string{"--literal-pathspecs", "ls-files", "-z", "--cached"}
	}
	if dir != "." {
		args = append(args, "--", dir+"/")
	}
	out, err := runGit(repo.root, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s dir %s: %w", side, dir, err)
	}
	var files []string
	for entry := range strings.SplitSeq(out, "\x00") {
		if entry != "" && path.Dir(entry) == dir {
			files = append(files, entry)
		}
	}
	return files, nil
}

func parseEntry(out, want string, idField int) (mode, id string, err error) {
	entries := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	if len(entries) != 1 || entries[0] == "" {
		return "", "", fmt.Errorf("want exactly one entry, got %d", len(entries))
	}
	// ls-tree prints "mode type id" and ls-files -s "mode id stage", so the id field differs.
	meta, got, ok := strings.Cut(entries[0], "\t")
	fields := strings.Fields(meta)
	if !ok || got != want || len(fields) <= idField {
		return "", "", fmt.Errorf("unexpected entry %q", entries[0])
	}
	return fields[0], fields[idField], nil
}
