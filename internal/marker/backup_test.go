package marker

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func seedFile(tb testing.TB, path, content string) {
	tb.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", path, err)
	}
}

func assertContent(tb testing.TB, path, want string) {
	tb.Helper()

	got, err := os.ReadFile(path) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		tb.Errorf("%q content = %q, want %q", path, got, want)
	}
}

func assertMissing(tb testing.TB, path string) {
	tb.Helper()

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("expected %q to be absent, stat err = %v", path, err)
	}
}

func assertSlotsIntact(tb testing.TB, seededContents map[string]string) {
	tb.Helper()

	for slot, content := range seededContents {
		assertContent(tb, slot, content)
	}
}

func TestBackup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedBackups []string
		wantSuffix  string
	}{
		{name: "free slot uses the bare suffix", wantSuffix: BackupSuffix},
		{
			name:        "bare slot taken falls back to .1",
			seedBackups: []string{BackupSuffix},
			wantSuffix:  BackupSuffix + ".1",
		},
		{
			name:        "bare and .1 taken falls back to .2",
			seedBackups: []string{BackupSuffix, BackupSuffix + ".1"},
			wantSuffix:  BackupSuffix + ".2",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "pre-commit")
			seedFile(t, path, "original")
			seededContents := make(map[string]string, len(testCase.seedBackups))
			for _, suffix := range testCase.seedBackups {
				seededContents[path+suffix] = "occupant " + suffix
				seedFile(t, path+suffix, seededContents[path+suffix])
			}

			backupPath, err := Backup(path)
			if err != nil {
				t.Fatalf("Backup: %v", err)
			}
			if want := path + testCase.wantSuffix; backupPath != want {
				t.Errorf("backupPath = %q, want %q", backupPath, want)
			}
			assertMissing(t, path)
			assertContent(t, backupPath, "original")
			assertSlotsIntact(t, seededContents)
		})
	}
}
