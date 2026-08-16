package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFileAtomic_RefusesSymlink is the reason this helper exists: nothing in
// the repo refused a symlink before, so a pre-placed link would have redirected a
// write onto a file outside the project.
func TestWriteFileAtomic_RefusesSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("original"), 0o600); err != nil {
		t.Fatalf("seed victim: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(victim, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	err := WriteFileAtomic(link, []byte("attacker"), 0o600)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("err = %v, want ErrSymlink", err)
	}
	body, readErr := os.ReadFile(victim)
	if readErr != nil {
		t.Fatalf("read victim: %v", readErr)
	}
	if string(body) != "original" {
		t.Errorf("victim was rewritten to %q", body)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		seed     string
		mode     fs.FileMode
		wantMode fs.FileMode
	}{
		{"fresh path", "", 0o644, 0o644},
		{"overwrite keeps the mode it is given", "old", 0o600, 0o600},
		{"executable mode survives", "", 0o755, 0o755},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "f.txt")
			if testCase.seed != "" {
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if err := WriteFileAtomic(path, []byte("new"), testCase.mode); err != nil {
				t.Fatalf("WriteFileAtomic: %v", err)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(body) != "new" {
				t.Errorf("body = %q, want %q", body, "new")
			}
			stat, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if stat.Mode().Perm() != testCase.wantMode {
				t.Errorf("mode = %v, want %v", stat.Mode().Perm(), testCase.wantMode)
			}
		})
	}
}

// TestWriteFileAtomic_LeavesNoStagingFile pins the cleanup: a directory holding
// a .staging-* leftover means a failed write littered the user's tree.
func TestWriteFileAtomic_LeavesNoStagingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := WriteFileAtomic(filepath.Join(dir, "f.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "f.txt" {
		t.Errorf("directory holds %d entries, want only f.txt", len(entries))
	}
}
