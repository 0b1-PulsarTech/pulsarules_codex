package vcs

import (
	"fmt"
	"path"
	"strings"
)

type TreeReader interface {
	HeadEntry(path string) (mode, id string, err error)
	StagedEntry(path string) (mode, id string, err error)
	HeadDir(dir string) ([]string, error)
	StagedDir(dir string) ([]string, error)
}

func (repo *repository) HeadEntry(path string) (mode, id string, err error) {
	var out string
	out, err = runGit(repo.root, "--literal-pathspecs", "ls-tree", "-z", "HEAD", "--", path)
	if err != nil {
		return "", "", fmt.Errorf("read HEAD entry %s: %w", path, err)
	}
	mode, id, err = parseEntry(out, path, 0, 2)
	if err != nil {
		return "", "", fmt.Errorf("read HEAD entry %s: %w", path, err)
	}
	return mode, id, nil
}

func (repo *repository) StagedEntry(path string) (mode, id string, err error) {
	var out string
	out, err = runGit(repo.root, "--literal-pathspecs", "ls-files", "-s", "-z", "--", path)
	if err != nil {
		return "", "", fmt.Errorf("read staged entry %s: %w", path, err)
	}
	mode, id, err = parseEntry(out, path, 0, 1)
	if err != nil {
		return "", "", fmt.Errorf("read staged entry %s: %w", path, err)
	}
	return mode, id, nil
}

func (repo *repository) HeadDir(dir string) ([]string, error) {
	args := []string{"--literal-pathspecs", "ls-tree", "-z", "--name-only", "HEAD"}
	return repo.listDir(dir, args, "HEAD")
}

func (repo *repository) StagedDir(dir string) ([]string, error) {
	args := []string{"--literal-pathspecs", "ls-files", "-z", "--cached"}
	return repo.listDir(dir, args, "staged")
}

func (repo *repository) listDir(dir string, args []string, side string) ([]string, error) {
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

func parseEntry(out, want string, modeField, idField int) (mode, id string, err error) {
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
	return fields[modeField], fields[idField], nil
}
