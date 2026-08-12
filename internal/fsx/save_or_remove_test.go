package fsx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveOrRemove(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		config      map[string]json.RawMessage
		wantRemoved bool
	}{
		{
			name:        "empty config removes the file",
			config:      map[string]json.RawMessage{},
			wantRemoved: true,
		},
		{
			name:        "non-empty config is saved",
			config:      map[string]json.RawMessage{"a": json.RawMessage(`"1"`)},
			wantRemoved: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}

			removed, err := SaveOrRemove(path, testCase.config)
			if err != nil {
				t.Fatalf("SaveOrRemove: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}

			_, statErr := os.Stat(path)
			gotExists := statErr == nil
			if gotExists == testCase.wantRemoved {
				t.Errorf("file exists = %v, want %v", gotExists, !testCase.wantRemoved)
			}
		})
	}
}

// TestSaveOrRemove_RemoveError covers the failure path: an empty config
// whose path is already gone must surface the remove error instead of
// silently reporting success.
func TestSaveOrRemove_RemoveError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.json")

	removed, err := SaveOrRemove(path, map[string]json.RawMessage{})
	if err == nil {
		t.Fatal("SaveOrRemove(missing path) err = nil, want a remove error")
	}
	if removed {
		t.Error("removed = true, want false on remove failure")
	}
}

// TestSaveOrRemove_SaveError covers the other failure path: a non-empty
// config whose parent directory is missing must surface the save error.
func TestSaveOrRemove_SaveError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing-parent", "config.json")

	removed, err := SaveOrRemove(path, map[string]json.RawMessage{"a": json.RawMessage(`"1"`)})
	if err == nil {
		t.Fatal("SaveOrRemove into a missing parent dir err = nil, want a save error")
	}
	if removed {
		t.Error("removed = true, want false on save failure")
	}
}
