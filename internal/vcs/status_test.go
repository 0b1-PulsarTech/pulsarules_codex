package vcs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorktreeStatus_CleanRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		t.Fatalf("WorktreeStatus: %v", err)
	}
	if len(status.Changes) != 0 {
		t.Fatalf("changes = %+v, want none", status.Changes)
	}
}

func stageMixedChanges(t *testing.T, dir string) {
	t.Helper()

	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{
		"modified.go":        "package a\n",
		"staged_modified.go": "package a\n",
		"deleted.go":         "package a\n",
		"staged_deleted.go":  "package a\n",
		"old_name.go":        "package a\n",
	})

	writeFile(t, dir, "modified.go", "package a // changed\n")

	writeFile(t, dir, "staged_modified.go", "package a // changed\n")
	runGitOrFatal(t, dir, "add", "staged_modified.go")

	mustRemove(t, dir, "deleted.go")

	mustRemove(t, dir, "staged_deleted.go")
	runGitOrFatal(t, dir, "add", "staged_deleted.go")

	runGitOrFatal(t, dir, "mv", "old_name.go", "new_name.go")

	writeFile(t, dir, "untracked.go", "package a\n")
}

func mustRemove(t *testing.T, dir, path string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, path)); err != nil {
		t.Fatalf("remove %s: %v", path, err)
	}
}

func TestWorktreeStatus_Changes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stageMixedChanges(t, dir)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		t.Fatalf("WorktreeStatus: %v", err)
	}

	byPath := make(map[string]Change, len(status.Changes))
	for _, change := range status.Changes {
		byPath[change.Path] = change
	}

	testCases := []struct {
		name        string
		path        string
		wantStaged  bool
		wantOldPath string
	}{
		{"unstaged modification", "modified.go", false, ""},
		{"staged modification", "staged_modified.go", true, ""},
		{"unstaged deletion", "deleted.go", false, ""},
		{"staged deletion", "staged_deleted.go", true, ""},
		{"untracked file", "untracked.go", false, ""},
		{"staged rename", "new_name.go", true, "old_name.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			change, ok := byPath[testCase.path]
			if !ok {
				t.Fatalf("path %q missing from status; got %+v", testCase.path, status.Changes)
			}
			if change.Staged != testCase.wantStaged {
				t.Fatalf("Staged = %v, want %v", change.Staged, testCase.wantStaged)
			}
			if change.OldPath != testCase.wantOldPath {
				t.Fatalf("OldPath = %q, want %q", change.OldPath, testCase.wantOldPath)
			}
			if change.Extension != ".go" {
				t.Fatalf("Extension = %q, want %q", change.Extension, ".go")
			}
		})
	}
}

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
