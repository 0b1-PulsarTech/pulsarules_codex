package marker

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackup asserts Backup renames an existing file to the bare
// ".pulsarules-backup" slot when that slot is free, and falls back to a
// numbered slot (.1, .2, ...) without ever overwriting a backup already
// occupying an earlier slot.
func TestBackup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedBackups []string // pre-existing backup slot suffixes, relative to path
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

			dir := t.TempDir()
			path := filepath.Join(dir, "pre-commit")
			if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
				t.Fatalf("seed original: %v", err)
			}
			seededContents := make(map[string]string, len(testCase.seedBackups))
			for _, suffix := range testCase.seedBackups {
				slot := path + suffix
				content := "occupant " + suffix
				if err := os.WriteFile(slot, []byte(content), 0o600); err != nil {
					t.Fatalf("seed backup slot %q: %v", slot, err)
				}
				seededContents[slot] = content
			}

			backupPath, err := Backup(path)
			if err != nil {
				t.Fatalf("Backup: %v", err)
			}
			want := path + testCase.wantSuffix
			if backupPath != want {
				t.Errorf("backupPath = %q, want %q", backupPath, want)
			}
			if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("expected original path to be gone, stat err = %v", statErr)
			}
			got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
			if readErr != nil {
				t.Fatalf("read backup: %v", readErr)
			}
			if string(got) != "original" {
				t.Errorf("backup content = %q, want %q", got, "original")
			}
			// Every earlier backup slot must survive untouched.
			for slot, content := range seededContents {
				got, readErr := os.ReadFile(slot) //nolint:gosec // test fixture.
				if readErr != nil {
					t.Fatalf("read seeded slot %q: %v", slot, readErr)
				}
				if string(got) != content {
					t.Errorf("seeded slot %q content = %q, want %q (clobbered)", slot, got, content)
				}
			}
		})
	}
}

// TestRestore asserts Restore renames the bare backup slot back to path and
// reports true, leaves a missing backup as a no-op reporting false, and never
// touches a numbered overflow slot.
func TestRestore(t *testing.T) {
	t.Parallel()

	t.Run("restores an existing backup", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "pre-commit")
		backupPath := path + BackupSuffix
		if err := os.WriteFile(backupPath, []byte("mine"), 0o600); err != nil {
			t.Fatalf("seed backup: %v", err)
		}

		restored, err := Restore(path)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if !restored {
			t.Fatal("restored = false, want true")
		}
		got, readErr := os.ReadFile(path) //nolint:gosec // test fixture.
		if readErr != nil {
			t.Fatalf("read restored path: %v", readErr)
		}
		if string(got) != "mine" {
			t.Errorf("restored content = %q, want %q", got, "mine")
		}
		if _, statErr := os.Stat(backupPath); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected backup slot to be gone, stat err = %v", statErr)
		}
	})

	t.Run("no backup is a no-op", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "pre-commit")
		restored, err := Restore(path)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if restored {
			t.Error("restored = true, want false for a missing backup")
		}
	})

	t.Run("leaves a numbered overflow backup alone", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "pre-commit")
		overflow := path + BackupSuffix + ".1"
		if err := os.WriteFile(overflow, []byte("overflow"), 0o600); err != nil {
			t.Fatalf("seed overflow backup: %v", err)
		}

		restored, err := Restore(path)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if restored {
			t.Error("restored = true, want false (only the bare slot is restored)")
		}
		if _, statErr := os.Stat(overflow); statErr != nil {
			t.Errorf("expected overflow backup to survive, stat err = %v", statErr)
		}
	})
}

// TestBackupMessage_RestoreMessage asserts the two formatters produce the
// phrasing every install/uninstall site reports verbatim.
func TestBackupMessage_RestoreMessage(t *testing.T) {
	t.Parallel()

	if got, want := BackupMessage(
		"/a/b",
		"/a/b.pulsarules-backup",
	), "backed up existing /a/b to /a/b.pulsarules-backup"; got != want {
		t.Errorf("BackupMessage = %q, want %q", got, want)
	}
	if got, want := RestoreMessage("/a/b"), "restored backup to /a/b"; got != want {
		t.Errorf("RestoreMessage = %q, want %q", got, want)
	}
}

// TestOrphans asserts only the NUMBERED slots are reported: Restore consumes the
// base one, so listing it would name a backup that is about to disappear.
func TestOrphans(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedBackups []string
		wantCount   int
	}{
		{name: "no backups at all"},
		{name: "base slot alone is not an orphan", seedBackups: []string{BackupSuffix}},
		{
			name:        "numbered slot is an orphan",
			seedBackups: []string{BackupSuffix, BackupSuffix + ".1"},
			wantCount:   1,
		},
		{
			name:        "consecutive numbered slots are all orphans",
			seedBackups: []string{BackupSuffix, BackupSuffix + ".1", BackupSuffix + ".2"},
			wantCount:   2,
		},
		{
			name:        "a gap stops the walk",
			seedBackups: []string{BackupSuffix, BackupSuffix + ".2"},
			wantCount:   0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "pre-commit")
			for _, suffix := range testCase.seedBackups {
				if err := os.WriteFile(path+suffix, []byte("x"), 0o600); err != nil {
					t.Fatalf("seed %q: %v", suffix, err)
				}
			}
			got, err := Orphans(path)
			if err != nil {
				t.Fatalf("Orphans: %v", err)
			}
			if len(got) != testCase.wantCount {
				t.Errorf("Orphans() = %v, want %d slot(s)", got, testCase.wantCount)
			}
			if len(got) > 0 && !strings.Contains(OrphanMessage(path, got), path) {
				t.Errorf("OrphanMessage does not name %q", path)
			}
		})
	}
}
