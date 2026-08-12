package agentswire

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveAgents_UserAuthored(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedContent string
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
			name:        "missing file with no seed content is a no-op",
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

func TestRemoveAgents_PathIsDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectDir, "AGENTS.md"), 0o750); err != nil {
		t.Fatalf("seed directory in place of AGENTS.md: %v", err)
	}

	if _, err := RemoveAgents(projectDir); err == nil {
		t.Error("expected an error reading a directory as AGENTS.md, got nil")
	}
}
