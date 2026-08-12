package cursorwire

import (
	"os"
	"path/filepath"
	"testing"
)

const seededBody = "---\ndescription: seed\nglobs:\nalwaysApply: false\n---\n" +
	"<!-- Installed by pulsarules_cli -->\n\nold body\n"

// TestWriteRule covers a fresh write, refreshing a file this tool wrote
// before, and refusing to overwrite a foreign file at the same path - the
// same three cases agentswire.TestWriteAgents_OwnershipDiscipline covers for
// AGENTS.md, since <id>.mdc is a name a user's own Cursor rule could
// plausibly already occupy.
func TestWriteRule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seedContent string
		wantWrote   bool
	}{
		{name: "fresh write", wantWrote: true},
		{name: "refreshes a tool-written file", seedContent: seededBody, wantWrote: true},
		{
			name:        "foreign file is not overwritten",
			seedContent: "# My own rule\nDo not touch.\n",
			wantWrote:   false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, RulesDir, "go-style.mdc")
			if testCase.seedContent != "" {
				if mkErr := os.MkdirAll(filepath.Dir(path), 0o750); mkErr != nil {
					t.Fatalf("mkdir: %v", mkErr)
				}
				if seedErr := os.WriteFile(
					path,
					[]byte(testCase.seedContent),
					0o600,
				); seedErr != nil {
					t.Fatalf("seed: %v", seedErr)
				}
			}

			newBody := "---\ndescription: new\nglobs:\nalwaysApply: false\n---\n" +
				"<!-- Installed by pulsarules_cli -->\n\nnew body\n"
			wrote, err := WriteRule(projectDir, "go-style", newBody)
			if err != nil {
				t.Fatalf("WriteRule: %v", err)
			}
			if wrote != testCase.wantWrote {
				t.Errorf("wrote = %v, want %v", wrote, testCase.wantWrote)
			}

			got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
			if readErr != nil {
				t.Fatalf("read %q: %v", path, readErr)
			}
			if testCase.wantWrote {
				if string(got) != newBody {
					t.Errorf("content = %q, want %q", got, newBody)
				}
			} else if string(got) != testCase.seedContent {
				t.Errorf("foreign content changed, got %q want %q", got, testCase.seedContent)
			}
		})
	}
}

// TestWriteRule_MkdirError asserts a project dir blocked by a file surfaces
// an error instead of silently doing nothing.
func TestWriteRule_MkdirError(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".cursor"), []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	if _, err := WriteRule(projectDir, "go-style", "body"); err == nil {
		t.Error("expected an error, got nil")
	}
}

// TestWriteRule_ReadError asserts a rule path occupied by a directory (so
// ReadFile fails with something other than not-exist) surfaces an error.
func TestWriteRule_ReadError(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	path := filepath.Join(projectDir, RulesDir, "go-style.mdc")
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("seed directory in place of go-style.mdc: %v", err)
	}

	if _, err := WriteRule(projectDir, "go-style", "body"); err == nil {
		t.Error("expected an error reading a directory as go-style.mdc, got nil")
	}
}
