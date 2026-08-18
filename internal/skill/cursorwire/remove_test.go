package cursorwire

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestRemoveRules covers the full contract: a mix of tool-written and
// foreign .mdc files in the same directory, a missing directory (never
// installed), and the directory being removed once emptied of every
// tool-written file.
func TestRemoveRules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        map[string]string // filename -> content; nil means no dir at all
		wantRemoved []string
		wantSurvive []string // filenames that must still exist afterward
	}{
		{
			name: "removes only tool-written files",
			seed: map[string]string{
				"go-style.mdc": seededBody,
				"foreign.mdc":  "# my own rule\n",
			},
			wantRemoved: []string{"go-style"},
			wantSurvive: []string{"foreign.mdc"},
		},
		{
			name: "removes the directory once emptied",
			seed: map[string]string{
				"go-style.mdc": seededBody,
			},
			wantRemoved: []string{"go-style"},
		},
		{
			name: "missing directory is a no-op",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			dir := filepath.Join(projectDir, RulesDir)
			if testCase.seed != nil {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				for name, content := range testCase.seed {
					if err := os.WriteFile(
						filepath.Join(dir, name),
						[]byte(content),
						0o600,
					); err != nil {
						t.Fatalf("seed %q: %v", name, err)
					}
				}
			}

			removed, err := RemoveRules(projectDir)
			if err != nil {
				t.Fatalf("RemoveRules: %v", err)
			}
			slices.Sort(removed)
			slices.Sort(testCase.wantRemoved)
			if !slices.Equal(removed, testCase.wantRemoved) {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}
			for _, name := range testCase.wantSurvive {
				if _, statErr := os.Stat(filepath.Join(dir, name)); statErr != nil {
					t.Errorf("expected %q to survive, stat err = %v", name, statErr)
				}
			}
			if len(testCase.wantSurvive) == 0 && testCase.seed != nil {
				if _, statErr := os.Stat(dir); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("expected %q removed once emptied, stat err = %v", dir, statErr)
				}
			}
		})
	}
}

// TestRemoveRules_Idempotent asserts running RemoveRules twice is not an
// error, and the second run removes nothing further.
func TestRemoveRules_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, RulesDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go-style.mdc"),
		[]byte(seededBody),
		0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, err := RemoveRules(projectDir)
	if err != nil || len(first) != 1 {
		t.Fatalf("RemoveRules #1: removed=%v err=%v", first, err)
	}
	second, err := RemoveRules(projectDir)
	if err != nil || len(second) != 0 {
		t.Fatalf("RemoveRules #2: removed=%v err=%v", second, err)
	}
}

// TestRemoveRules_ToleratesVanishedFile pins the ReadDir -> ReadFile race: ownsExisting answers
// true for an absent path, which is right for WriteRule's "safe to create" and wrong for a delete.
// A dangling symlink reproduces it deterministically - listed like any .mdc, unreadable when
// checked - and os.Remove deletes a symlink without resolving it, so the old code deleted and
// reported an entry whose marker it never verified. RemoveRules must leave what it cannot verify.
func TestRemoveRules_ToleratesVanishedFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, RulesDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ghost := filepath.Join(dir, "ghost.mdc")
	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), ghost); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "go-style.mdc"),
		[]byte(seededBody),
		0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	removed, err := RemoveRules(projectDir)
	if err != nil {
		t.Fatalf("RemoveRules: %v", err)
	}
	if !slices.Contains(removed, "go-style") {
		t.Errorf("expected go-style removed, got %v", removed)
	}
	if slices.Contains(removed, "ghost") {
		t.Errorf("expected the vanished ghost file not reported as removed, got %v", removed)
	}
}

// TestRemoveRules_ReadDirError asserts a listing failure on the parent
// directory (occupied by a file, not a dir) surfaces an error.
func TestRemoveRules_ReadDirError(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".cursor"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	if _, err := RemoveRules(projectDir); err == nil {
		t.Error("expected an error, got nil")
	}
}
