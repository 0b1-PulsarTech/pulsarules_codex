package vcs

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if !status.IsClean() {
		t.Fatalf("IsClean() = false, want true; changes = %+v", status.Changes)
	}
}

// stageMixedChanges commits a fixed set of files, then leaves dir with one
// of each kind of worktree change TestWorktreeStatus_Changes asserts on.
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

// mustRemove deletes path from dir on disk, failing the test on error.
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
	for _, c := range status.Changes {
		byPath[c.Path] = c
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

	if status.IsClean() {
		t.Fatal("IsClean() = true, want false")
	}
	if exts := status.Extensions(); !exts[".go"] {
		t.Fatalf("Extensions() = %v, want it to contain .go", exts)
	}

	rendered := status.String()
	if rendered == "" {
		t.Fatal("String() = empty, want a rendered listing")
	}
	wantLine := "R  old_name.go -> new_name.go"
	if !slices.Contains(strings.Split(rendered, "\n"), wantLine) {
		t.Fatalf("String() = %q, want it to contain %q", rendered, wantLine)
	}
}

func TestStatus_IsCleanAndExtensions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		status      Status
		wantClean   bool
		wantExts    map[string]bool
		wantNonZero bool
	}{
		{
			name:      "no changes",
			status:    Status{},
			wantClean: true,
			wantExts:  map[string]bool{},
		},
		{
			name: "mixed extensions",
			status: Status{Changes: []Change{
				{Path: "a.go", Extension: ".go", Staging: ' ', Worktree: 'M'},
				{Path: "b.sql", Extension: ".sql", Staging: 'M', Worktree: ' '},
				{Path: "c.go", Extension: ".go", Staging: ' ', Worktree: 'M'},
			}},
			wantClean:   false,
			wantExts:    map[string]bool{".go": true, ".sql": true},
			wantNonZero: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.status.IsClean(); got != testCase.wantClean {
				t.Fatalf("IsClean() = %v, want %v", got, testCase.wantClean)
			}
			got := testCase.status.Extensions()
			if len(got) != len(testCase.wantExts) {
				t.Fatalf("Extensions() = %v, want %v", got, testCase.wantExts)
			}
			for ext := range testCase.wantExts {
				if !got[ext] {
					t.Fatalf("Extensions() = %v, missing %q", got, ext)
				}
			}
			if s := testCase.status.String(); testCase.wantNonZero && s == "" {
				t.Fatal("String() = empty, want non-empty")
			}
		})
	}
}

// writeFile writes content at path inside dir, creating parent directories
// as needed.
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
