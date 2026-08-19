package ownedfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

func TestRestore_RestoresAnExistingBackup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pre-commit")
	backupPath := path + BackupSuffix
	seedFile(t, backupPath, "mine")

	restored, err := Restore(path)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if !restored {
		t.Fatal("restored = false, want true")
	}
	assertContent(t, path, "mine")
	assertMissing(t, backupPath)
}

func TestRestore_NoBackupIsANoOp(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pre-commit")
	restored, err := Restore(path)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored {
		t.Error("restored = true, want false for a missing backup")
	}
}

func TestRestore_NumberedOverflowBackup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pre-commit")
	overflow := path + BackupSuffix + ".1"
	seedFile(t, overflow, "overflow")

	restored, err := Restore(path)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored {
		t.Error("restored = true, want false (only the bare slot is restored)")
	}
	assertContent(t, overflow, "overflow")
}

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

func TestOrphansExcludesTheBaseSlotRestoreConsumes(t *testing.T) {
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
			name:        "a slot past a numbering gap is still named",
			seedBackups: []string{BackupSuffix, BackupSuffix + ".2"},
			wantCount:   1,
		},
		{
			name: "a non-numeric sibling is not a slot",
			seedBackups: []string{
				BackupSuffix + ".1", BackupSuffix + ".notes", BackupSuffix + ".007",
			},
			wantCount: 1,
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

func TestOrphans_MissingDir(t *testing.T) {
	t.Parallel()

	got, err := Orphans(filepath.Join(t.TempDir(), "no-such-dir", "pre-commit"))
	if err != nil {
		t.Fatalf("Orphans on a missing dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Orphans() = %v, want none", got)
	}
}

func TestOrphans_UnreadableDir(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := Orphans(filepath.Join(filePath, "pre-commit")); err == nil {
		t.Fatal("Orphans through a non-dir parent: expected an error")
	}
}
