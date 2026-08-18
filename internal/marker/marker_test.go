package marker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		content    string
		missing    bool
		wantExists bool
		wantOurs   bool
	}{
		{name: "absent file", missing: true, wantExists: false, wantOurs: false},
		{
			name:       "foreign content",
			content:    "hand-authored notes",
			wantExists: true,
			wantOurs:   false,
		},
		{
			name:       "our content",
			content:    "# " + Installed + "\nbody",
			wantExists: true,
			wantOurs:   true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "asset")
			if !testCase.missing {
				if err := os.WriteFile(path, []byte(testCase.content), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}

			exists, ours, err := Check(path)
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if exists != testCase.wantExists || ours != testCase.wantOurs {
				t.Errorf(
					"Check(%q) = (%v, %v), want (%v, %v)",
					path, exists, ours, testCase.wantExists, testCase.wantOurs,
				)
			}
		})
	}
}

// TestCheck_ReadError covers the failure path: a path that exists but cannot
// be read as a file (here, a directory) must surface an error rather than
// silently reporting "not ours".
func TestCheck_ReadError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, _, err := Check(dir); err == nil {
		t.Fatal("Check(dir) err = nil, want a read error")
	}
}
