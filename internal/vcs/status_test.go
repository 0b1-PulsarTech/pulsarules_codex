package vcs

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
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

	assertStatusSummary(t, status)
}

func assertStatusSummary(t *testing.T, status Status) {
	t.Helper()

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

func TestWorktreeStatus_RealPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{
		"old name.go": "package a\n// one two three four five six seven\n",
		"src.go":      "package a\n// alpha beta gamma delta epsilon zeta eta\n",
	})
	runGitOrFatal(t, dir, "mv", "old name.go", "new name.go")
	for _, path := range []string{
		"sp ace.go", "tab\tname.go", "caf\u00e9.go", "quo\"te.go", "a -> b.go",
		"newpkg/sub/deep.go",
	} {
		writeFile(t, dir, path, "package a\n")
	}

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
		wantOldPath string
	}{
		{"space", "sp ace.go", ""},
		{"tab", "tab\tname.go", ""},
		{"non-ascii", "caf\u00e9.go", ""},
		{"quote", "quo\"te.go", ""},
		{"arrow", "a -> b.go", ""},
		{"nested untracked file", "newpkg/sub/deep.go", ""},
		{"rename with space", "new name.go", "old name.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			change, ok := byPath[testCase.path]
			if !ok {
				t.Fatalf("path %q missing; got %+v", testCase.path, status.Changes)
			}
			if change.OldPath != testCase.wantOldPath {
				t.Fatalf("OldPath = %q, want %q", change.OldPath, testCase.wantOldPath)
			}
		})
	}
	if _, ok := byPath["newpkg/"]; ok {
		t.Fatal("untracked directory listed as one entry")
	}
}

func TestWorktreeStatus_StagedCopy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	body := "package a\n// alpha beta gamma delta epsilon zeta eta theta iota\n"
	writeAndCommit(t, dir, "initial", map[string]string{"src.go": body})
	writeFile(t, dir, "src.go", body+"// edit\n")
	writeFile(t, dir, "dup.go", body)
	runGitOrFatal(t, dir, "add", "-A")
	runGitOrFatal(t, dir, "config", "status.renames", "copies")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		t.Fatalf("WorktreeStatus: %v", err)
	}
	found := slices.ContainsFunc(status.Changes, func(change Change) bool {
		return change.Path == "dup.go" && change.OldPath == "src.go" && change.Staging == 'C'
	})
	if !found {
		t.Fatalf("copy dup.go <- src.go missing; parsed %+v", status.Changes)
	}
}

func TestStatus_StringRoutesPathsThroughEscape(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		oldPath string
		want    string
	}{
		{"arrow in untracked name", "a -> b.go", "", "?? \"a -> b.go\""},
		{"arrow in old path", "y.go", "x -> m.go", "?? \"x -> m.go\" -> y.go"},
		{"rename with newline in new path", "n\new.go", "old.go", "?? old.go -> \"n\\new.go\""},
		{"rename with newline in old path", "new.go", "o\nld.go", "?? \"o\\nld.go\" -> new.go"},
		{"plain path", "a.go", "", "?? a.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			change := Change{
				Path:     testCase.path,
				OldPath:  testCase.oldPath,
				Staging:  '?',
				Worktree: '?',
			}
			if got := (Status{Changes: []Change{change}}).String(); got != testCase.want {
				t.Fatalf("String() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestWorktreeStatus_LeavesIndexUntouched(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "a.go"), later, later); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	indexPath := filepath.Join(dir, ".git", "index")
	before, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("stat index: %v", err)
	}
	beforeBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err = repo.WorktreeStatus(); err != nil {
		t.Fatalf("WorktreeStatus: %v", err)
	}
	after, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("stat index: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("index mtime %v -> %v, want unchanged", before.ModTime(), after.ModTime())
	}
	afterBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatalf("index bytes changed (%d -> %d)", len(beforeBytes), len(afterBytes))
	}
}
