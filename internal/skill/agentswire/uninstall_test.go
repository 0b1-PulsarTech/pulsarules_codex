package agentswire

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestRemoveAgents covers the full contract: a tool-written AGENTS.md is
// removed, a pre-existing user-authored AGENTS.md (the highest-risk case,
// since it lives at a name a user very plausibly owns already) survives
// untouched, a missing file is a no-op, and re-running after removal is
// still a no-op.
func TestRemoveAgents(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedContent string // "" means no file at all
		wantRemoved bool
	}{
		{
			name:        "tool-written file is removed",
			seedContent: "# AGENTS.md\n\n<!-- Installed by pulsarules_cli -->\n",
			wantRemoved: true,
		},
		{
			name:        "user-authored file survives",
			seedContent: "# My own AGENTS.md\nDo not touch.\n",
			wantRemoved: false,
		},
		{
			name:        "missing file is a no-op",
			seedContent: "",
			wantRemoved: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, "AGENTS.md")
			if testCase.seedContent != "" {
				if err := os.WriteFile(path, []byte(testCase.seedContent), 0o600); err != nil {
					t.Fatalf("seed AGENTS.md: %v", err)
				}
			}

			removed, err := RemoveAgents(projectDir)
			if err != nil {
				t.Fatalf("RemoveAgents: %v", err)
			}
			if removed != testCase.wantRemoved {
				t.Errorf("removed = %v, want %v", removed, testCase.wantRemoved)
			}

			_, statErr := os.Stat(path)
			switch {
			case testCase.seedContent == "":
				// nothing to assert; there was never a file
			case testCase.wantRemoved:
				if !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("expected %q removed, stat err = %v", path, statErr)
				}
			default:
				if statErr != nil {
					t.Errorf("expected %q to survive, stat err = %v", path, statErr)
				}
			}
		})
	}
}

// TestRemoveAgents_Idempotent asserts running RemoveAgents twice against a
// tool-written file is not an error.
func TestRemoveAgents_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	path := filepath.Join(projectDir, "AGENTS.md")
	content := "# AGENTS.md\n\n<!-- Installed by pulsarules_cli -->\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}

	if removed, err := RemoveAgents(projectDir); err != nil || !removed {
		t.Fatalf("RemoveAgents #1: removed=%v err=%v", removed, err)
	}
	if removed, err := RemoveAgents(projectDir); err != nil || removed {
		t.Fatalf("RemoveAgents #2: removed=%v err=%v", removed, err)
	}
}

// TestRemoveAgents_ReadError asserts a path AGENTS.md occupied by a directory
// (so ReadFile fails with something other than not-exist) surfaces an error.
func TestRemoveAgents_ReadError(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectDir, "AGENTS.md"), 0o750); err != nil {
		t.Fatalf("seed directory in place of AGENTS.md: %v", err)
	}

	if _, err := RemoveAgents(projectDir); err == nil {
		t.Error("expected an error reading a directory as AGENTS.md, got nil")
	}
}
