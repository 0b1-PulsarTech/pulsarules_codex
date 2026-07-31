package gitignore

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestEnsure covers creating the file, idempotency when every entry is
// already present, and appending missing entries while preserving existing
// content (with or without a trailing newline).
func TestEnsure(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		hasSeed bool
		seed    string
		entries []string
		wantHas []string
	}{
		{
			name:    "created when absent",
			entries: []string{"/bin/", "/hooks/"},
			wantHas: []string{"/bin/", "/hooks/"},
		},
		{
			name:    "idempotent when all entries present",
			hasSeed: true,
			seed:    "/bin/\n/hooks/\n",
			entries: []string{"/bin/", "/hooks/"},
			wantHas: []string{"/bin/", "/hooks/"},
		},
		{
			name:    "appended preserving existing lines",
			hasSeed: true,
			seed:    "foo\n",
			entries: []string{"/bin/"},
			wantHas: []string{"foo", "/bin/"},
		},
		{
			name:    "appended when no trailing newline",
			hasSeed: true,
			seed:    "foo",
			entries: []string{"/bin/"},
			wantHas: []string{"foo", "/bin/"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), "target")
			path := filepath.Join(dir, ".gitignore")
			if testCase.hasSeed {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure: %v", err)
			}
			for _, want := range testCase.wantHas {
				if !slices.Contains(splitLines(readFile(t, path)), want) {
					t.Errorf("gitignore %q missing %q", readFile(t, path), want)
				}
			}
			// Running twice must not duplicate an entry.
			if err := Ensure(dir, testCase.entries...); err != nil {
				t.Fatalf("Ensure (rerun): %v", err)
			}
			for _, entry := range testCase.entries {
				if n := strings.Count(readFile(t, path), entry); n != 1 {
					t.Errorf(
						"expected exactly one %q entry, got %d in %q",
						entry,
						n,
						readFile(t, path),
					)
				}
			}
		})
	}
}

func readFile(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}
