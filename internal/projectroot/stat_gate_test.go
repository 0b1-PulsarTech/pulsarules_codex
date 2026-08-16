package projectroot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatGate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		size int
		kind string
		want string
	}{
		{"small regular file", 10, "file", ""},
		{"exactly at the cap", MaxEditedFileBytes, "file", ""},
		{"one byte over the cap", MaxEditedFileBytes + 1, "file", "larger than the size cap"},
		{"missing", 0, "missing", "cannot stat"},
		{"directory", 0, "dir", "not a regular file"},
		{"symlink", 10, "link", "is a symlink"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "target")
			switch testCase.kind {
			case "file":
				if err := os.WriteFile(path, make([]byte, testCase.size), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			case "dir":
				if err := os.Mkdir(path, 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			case "link":
				realPath := path + "-real"
				if err := os.WriteFile(realPath, make([]byte, testCase.size), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
				if err := os.Symlink(realPath, path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			if got := StatGate(path); got != testCase.want {
				t.Errorf("StatGate = %q, want %q", got, testCase.want)
			}
		})
	}
}
