package tsalias

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiskReader(t *testing.T) {
	t.Parallel()

	fallback := func(name string) ([]byte, bool) {
		if name == "memory.json" {
			return []byte("from fallback"), true
		}
		return nil, false
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.json"), []byte("on disk"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	big := strings.Repeat(" ", maxConfigBytes+1)
	if err := os.WriteFile(filepath.Join(root, "big.json"), []byte(big), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir.json"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testCases := []struct {
		name, root, file, want string
		found                  bool
	}{
		{name: "regular file", root: root, file: "a.json", want: "on disk", found: true},
		{
			name:  "memory fallback",
			root:  root,
			file:  "memory.json",
			want:  "from fallback",
			found: true,
		},
		{name: "missing", root: root, file: "none.json"},
		{name: "over the cap", root: root, file: "big.json"},
		{name: "directory", root: root, file: "dir.json"},
		{name: "no root", file: "memory.json", want: "from fallback", found: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			done := make(chan struct{})
			var got []byte
			var found bool
			go func() {
				defer close(done)
				got, found = DiskReader(testCase.root, fallback)(testCase.file)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("read blocked")
			}
			if found != testCase.found || string(got) != testCase.want {
				t.Fatalf("read = %q, %v", got, found)
			}
		})
	}
}
