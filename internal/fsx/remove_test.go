package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveEmptyDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		missing    bool
		withEntry  bool
		wantExists bool
	}{
		{name: "missing dir is a no-op", missing: true, wantExists: false},
		{name: "empty dir is removed", wantExists: false},
		{name: "non-empty dir survives", withEntry: true, wantExists: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			dir := filepath.Join(base, "target")
			if !testCase.missing {
				if err := os.Mkdir(dir, 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			}
			if testCase.withEntry {
				if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("x"), 0o600); err != nil {
					t.Fatalf("seed entry: %v", err)
				}
			}

			if err := RemoveEmptyDir(dir); err != nil {
				t.Fatalf("RemoveEmptyDir: %v", err)
			}
			_, statErr := os.Stat(dir)
			gotExists := statErr == nil
			if gotExists != testCase.wantExists {
				t.Errorf(
					"dir exists = %v, want %v (stat err = %v)",
					gotExists,
					testCase.wantExists,
					statErr,
				)
			}
		})
	}
}

// TestRemoveEmptyDir_ReadError covers the failure path: a path that exists
// but cannot be listed as a directory (here, a plain file) must surface an
// error.
func TestRemoveEmptyDir_ReadError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := RemoveEmptyDir(path); err == nil {
		t.Fatal("RemoveEmptyDir(file) err = nil, want a read error")
	}
}
