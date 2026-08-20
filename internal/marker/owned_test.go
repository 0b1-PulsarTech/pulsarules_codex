package marker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ownedBody = "# " + Installed + "\nbody\n"

// TestInstallFile pins the ownership split: a foreign occupant is backed up,
// an owned one is replaced with the mode applied fresh, and an absent path is
// a plain write.
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
				if err := os.WriteFile(path, []byte(testCase.seed), testCase.seedMode); err != nil {
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

// TestUninstallFile pins the removal split: only an owned file is removed, and
// its removal restores the base backup slot when one is there.
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
			if testCase.seed != "" {
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if testCase.seedBackup != "" {
				backupSeedErr := os.WriteFile(
					path+BackupSuffix, []byte(testCase.seedBackup), 0o600,
				)
				if backupSeedErr != nil {
					t.Fatalf("seed backup: %v", backupSeedErr)
				}
			}
			removed, note, err := UninstallFile(path)
			if err != nil {
				t.Fatalf("UninstallFile: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			if gotNote := strings.Contains(note, path); gotNote != testCase.wantNote {
				t.Errorf("note = %q, want restore note %v", note, testCase.wantNote)
			}
			got, readErr := os.ReadFile(path)
			if testCase.wantContent == "" {
				if readErr == nil {
					t.Errorf("path still holds %q, want it gone", got)
				}
				return
			}
			if readErr != nil || string(got) != testCase.wantContent {
				t.Errorf("content = %q (%v), want %q", got, readErr, testCase.wantContent)
			}
		})
	}
}
