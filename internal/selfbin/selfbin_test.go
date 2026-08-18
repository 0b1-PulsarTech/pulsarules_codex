package selfbin

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopy covers a successful nested-destination copy and the
// failure path when the destination's parent cannot be created.
func TestCopy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		dst     func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "copies to a nested destination",
			dst: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nested", "bin", "copy")
			},
		},
		{
			name: "mkdir fails when the parent path is a file",
			dst: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				blocker := filepath.Join(dir, "blocker")
				if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
					t.Fatalf("seed blocker: %v", err)
				}
				return filepath.Join(blocker, "copy")
			},
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dst := testCase.dst(t)
			err := Copy(dst)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Copy: %v", err)
			}
			stat, statErr := os.Stat(dst)
			if statErr != nil {
				t.Fatalf("stat dst: %v", statErr)
			}
			if stat.Mode()&0o111 == 0 {
				t.Errorf("copied binary not executable: mode %v", stat.Mode())
			}
		})
	}
}
