package ownedfile

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const ownedBody = "# " + Marker + "\nbody\n"

func TestInstallFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		seed       string
		seedMode   os.FileMode
		wantNote   bool
		wantBackup bool
	}{
		{name: "absent path is a plain write"},
		{
			name:       "foreign occupant is backed up",
			seed:       "user content\n",
			seedMode:   0o600,
			wantNote:   true,
			wantBackup: true,
		},
		{
			name:     "owned occupant is replaced without a backup",
			seed:     ownedBody,
			seedMode: 0o600,
		},
		{
			name:     "read-only owned occupant is still replaced",
			seed:     ownedBody,
			seedMode: 0o500,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "asset")
			if testCase.seed != "" {
				if err := os.WriteFile(
					path, []byte(testCase.seed), testCase.seedMode,
				); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			note, err := InstallFile(path, []byte(ownedBody), 0o600)
			if err != nil {
				t.Fatalf("InstallFile: %v", err)
			}
			if gotNote := note != ""; gotNote != testCase.wantNote {
				t.Errorf("note = %q, want note %v", note, testCase.wantNote)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read back: %v", err)
			}
			if string(got) != ownedBody {
				t.Errorf("content = %q, want the installed body", got)
			}
			_, statErr := os.Lstat(path + BackupSuffix)
			if gotBackup := statErr == nil; gotBackup != testCase.wantBackup {
				t.Errorf("backup exists = %v, want %v", gotBackup, testCase.wantBackup)
			}
		})
	}
}

func seedAsset(tb testing.TB, path, seed, seedBackup string) {
	tb.Helper()

	if seed != "" {
		seedFile(tb, path, seed)
	}
	if seedBackup != "" {
		seedFile(tb, path+BackupSuffix, seedBackup)
	}
}

func assertAssetContent(tb testing.TB, path, want string) {
	tb.Helper()

	got, err := os.ReadFile(path)
	if want == "" {
		if err == nil {
			tb.Errorf("path still holds %q, want it gone", got)
		}
		return
	}
	if err != nil || string(got) != want {
		tb.Errorf("content = %q (%v), want %q", got, err, want)
	}
}

func TestUninstallFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		seedBackup  string
		wantRemoved bool
		wantNote    bool
		wantContent string
	}{
		{name: "absent path is a no-op"},
		{
			name:        "foreign file survives untouched",
			seed:        "user content\n",
			wantContent: "user content\n",
		},
		{
			name:        "owned file is removed",
			seed:        ownedBody,
			wantRemoved: true,
		},
		{
			name:        "removal restores the uncovered backup",
			seed:        ownedBody,
			seedBackup:  "previous hook\n",
			wantRemoved: true,
			wantNote:    true,
			wantContent: "previous hook\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "asset")
			seedAsset(t, path, testCase.seed, testCase.seedBackup)
			removed, note, err := uninstallFile(path)
			if err != nil {
				t.Fatalf("uninstallFile: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			if gotNote := strings.Contains(note, path); gotNote != testCase.wantNote {
				t.Errorf("note = %q, want restore note %v", note, testCase.wantNote)
			}
			assertAssetContent(t, path, testCase.wantContent)
		})
	}
}

func TestUninstallFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ours := filepath.Join(dir, "ours")
	restored := filepath.Join(dir, "restored")
	foreign := filepath.Join(dir, "foreign")
	absent := filepath.Join(dir, "absent")
	for path, body := range map[string]string{
		ours:                       ownedBody,
		restored:                   ownedBody,
		restored + BackupSuffix:    "user original\n",
		foreign:                    "user content\n",
		ours + BackupSuffix + ".1": "numbered slot\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	removed, ledger, err := UninstallFiles(wrapTestFailure, ours, restored, foreign, absent)
	if err != nil {
		t.Fatalf("UninstallFiles: %v", err)
	}
	if !slices.Equal(removed, []string{ours, restored}) {
		t.Errorf("removed = %q, want ours and restored", removed)
	}
	if !slices.Equal(ledger.Restored, []string{RestoreMessage(restored)}) {
		t.Errorf("Restored = %q, want one restore note", ledger.Restored)
	}
	wantOrphan := OrphanMessage(ours, []string{ours + BackupSuffix + ".1"})
	if !slices.Equal(ledger.Orphaned, []string{wantOrphan}) {
		t.Errorf("Orphaned = %q, want %q", ledger.Orphaned, wantOrphan)
	}
	if _, statErr := os.Stat(foreign); statErr != nil {
		t.Errorf("foreign file must survive: %v", statErr)
	}
}

func wrapTestFailure(path string, err error) error {
	return fmt.Errorf("uninstall %q: %w", filepath.Base(path), err)
}

func TestUninstallFiles_CheckFails(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, _, err := UninstallFiles(wrapTestFailure, filepath.Join(blocker, "hook"))
	if err == nil || !strings.HasPrefix(err.Error(), `uninstall "hook": check `) {
		t.Fatalf("UninstallFiles err = %v, want the caller's wording over a check error", err)
	}
}

func TestUninstallFiles_OrphanScanFailsBare(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hook")
	if err := os.WriteFile(path, []byte(ownedBody), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(dir, 0o300); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	_, _, err := UninstallFiles(wrapTestFailure, path)
	if err == nil || !strings.HasPrefix(err.Error(), "list backups of ") {
		t.Fatalf("UninstallFiles err = %v, want the scan failure without the caller's wording", err)
	}
}

func TestRestoreVacated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vacated := filepath.Join(dir, "vacated")
	standing := filepath.Join(dir, "standing")
	bare := filepath.Join(dir, "bare")
	for path, body := range map[string]string{
		vacated + BackupSuffix:        "user original\n",
		vacated + BackupSuffix + ".1": "numbered slot\n",
		standing:                      "user file now\n",
		standing + BackupSuffix:       "must stay a backup\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	ledger, err := RestoreVacated(vacated, standing, bare)
	if err != nil {
		t.Fatalf("RestoreVacated: %v", err)
	}
	if !slices.Equal(ledger.Restored, []string{RestoreMessage(vacated)}) {
		t.Errorf("Restored = %q, want only the vacated path", ledger.Restored)
	}
	wantOrphans := []string{OrphanMessage(vacated, []string{vacated + BackupSuffix + ".1"})}
	if !slices.Equal(ledger.Orphaned, wantOrphans) {
		t.Errorf("Orphaned = %q, want %q", ledger.Orphaned, wantOrphans)
	}
	if got, readErr := os.ReadFile(standing); readErr != nil || string(got) != "user file now\n" {
		t.Errorf("standing file = %q (%v), want it untouched", got, readErr)
	}
	if _, statErr := os.Stat(standing + BackupSuffix); statErr != nil {
		t.Errorf("the standing path's backup must stay in place: %v", statErr)
	}
}

func TestRestoreVacated_RestoreFails(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "vacated")
	if err := os.WriteFile(path+BackupSuffix, []byte("user original\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	_, err := RestoreVacated(path)
	if err == nil || !strings.HasPrefix(err.Error(), "restore backup: restore ") {
		t.Fatalf("RestoreVacated err = %v, want the restore failure worded as base did", err)
	}
}
