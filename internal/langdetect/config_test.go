package langdetect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		set   Set
		write bool
	}{
		{name: "no config written yet", write: false},
		{name: "empty set persists as empty", set: NewSet(), write: true},
		{name: "single language", set: NewSet(Go), write: true},
		{
			name:  "multi-language set",
			set:   NewSet(Dart, Svelte, TypeScript),
			write: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repoDir := t.TempDir()
			if testCase.write {
				if err := SaveConfig(repoDir, testCase.set); err != nil {
					t.Fatalf("SaveConfig: %v", err)
				}
			}

			got, ok, err := LoadConfig(repoDir)
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if ok != testCase.write {
				t.Fatalf("LoadConfig ok = %v, want %v", ok, testCase.write)
			}
			if !ok {
				return
			}
			if want, gotStrings := testCase.set.Strings(), got.Strings(); !slices.Equal(
				gotStrings,
				want,
			) {
				t.Errorf("LoadConfig = %v, want %v", gotStrings, want)
			}
		})
	}
}

func TestRemoveConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		write     bool
		wantOK    bool
		wantFound bool
	}{
		{name: "removes an existing config", write: true, wantOK: true, wantFound: true},
		{name: "no-op when nothing was written", write: false, wantOK: true, wantFound: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repoDir := t.TempDir()
			if testCase.write {
				if err := SaveConfig(repoDir, NewSet(Go)); err != nil {
					t.Fatalf("SaveConfig: %v", err)
				}
			}

			removed, err := RemoveConfig(repoDir)
			if (err == nil) != testCase.wantOK {
				t.Fatalf("RemoveConfig err = %v, wantOK %v", err, testCase.wantOK)
			}
			if removed != testCase.wantFound {
				t.Errorf("RemoveConfig removed = %v, want %v", removed, testCase.wantFound)
			}
			if _, statErr := os.Stat(
				filepath.Join(repoDir, ConfigFileName),
			); !os.IsNotExist(
				statErr,
			) {
				t.Errorf("expected %s to be gone, stat err = %v", ConfigFileName, statErr)
			}
		})
	}
}

func TestSaveConfig_Overwrites(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := SaveConfig(repoDir, NewSet(Go)); err != nil {
		t.Fatalf("SaveConfig #1: %v", err)
	}
	if err := SaveConfig(repoDir, NewSet(Dart)); err != nil {
		t.Fatalf("SaveConfig #2: %v", err)
	}

	got, ok, err := LoadConfig(repoDir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !ok {
		t.Fatal("LoadConfig ok = false, want true")
	}
	if want := []string{"dart"}; !slices.Equal(got.Strings(), want) {
		t.Errorf("LoadConfig = %v, want %v", got.Strings(), want)
	}
}
