package vcs

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

func (repo *Repository) Entry(side Side, file string) (mode, id string, err error) {
	args := []string{"--literal-pathspecs", "ls-tree", "-z", "HEAD", "--", file}
	idField := 2
	if side == Staged {
		args = []string{"--literal-pathspecs", "ls-files", "-s", "-z", "--", file}
		idField = 1
	}
	var out string
	out, err = runGit(repo.root, args...)
	if err != nil {
		return "", "", fmt.Errorf("read %s entry %s: %w", side, file, err)
	}
	mode, id, err = parseEntry(out, file, idField)
	if err != nil {
		return "", "", fmt.Errorf("read %s entry %s: %w", side, file, err)
	}
	return mode, id, nil
}

func (repo *Repository) Dir(side Side, dir string) ([]string, error) {
	entries, err := repo.list(side, dir)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(
		entries,
		func(entry string) bool { return path.Dir(entry) != dir },
	), nil
}

func (repo *Repository) Files(side Side, dir string) ([]string, error) {
	return repo.list(side, dir, "-r")
}

func (repo *Repository) list(side Side, dir string, headFlags ...string) ([]string, error) {
	args := append([]string{"--literal-pathspecs", "ls-tree", "-z", "--name-only"}, headFlags...)
	args = append(args, "HEAD")
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
	return slices.DeleteFunc(strings.Split(out, "\x00"), func(entry string) bool {
		return entry == ""
	}), nil
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
