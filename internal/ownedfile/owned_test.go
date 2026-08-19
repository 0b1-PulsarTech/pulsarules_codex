package ownedfile

import (
	"os"
	"path/filepath"
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
			assertAssetContent(t, path, testCase.wantContent)
		})
	}
}
