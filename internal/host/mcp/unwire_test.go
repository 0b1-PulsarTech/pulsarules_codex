package mcp

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestRemoveMCP_DeletesOurFile(t *testing.T) {
	fakeOnPath(t, "gopls")

	repoDir := t.TempDir()
	if _, _, err := Write(
		repoDir,
		detect.NewSet(language.Go),
	); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}

	if _, err := Remove(repoDir); err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".mcp.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected .mcp.json to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".gitignore")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected .gitignore to be removed once empty, stat err = %v", err)
	}
}

func TestRemoveMCP_PreservesUnrelated(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	seed := `{
  "someTopLevel": {"keep": true},
  "mcpServers": {
    "other": {"command": "other-mcp", "args": ["serve"], "cwd": "/x"},
    "gopls": {"command": "gopls", "args": ["mcp"], "cwd": "` + repoDir + `"}
  }
}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := Remove(repoDir); err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var config struct {
		SomeTopLevel json.RawMessage            `json:"someTopLevel"`
		MCPServers   map[string]json.RawMessage `json:"mcpServers"`
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(config.SomeTopLevel) == 0 {
		t.Error("lost unrelated top-level key someTopLevel")
	}
	if _, ok := config.MCPServers["gopls"]; ok {
		t.Error("gopls server should have been removed")
	}
	if _, ok := config.MCPServers["other"]; !ok {
		t.Error("lost unrelated server other")
	}
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestRemoveMCP_StripsGitignore(t *testing.T) {
	fakeOnPath(t, "gopls")

	repoDir := t.TempDir()
	seed := `{"mcpServers": {"myserver": {"command": "myserver"}}}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, _, err := Write(
		repoDir,
		detect.NewSet(language.Go),
	); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	giPath := filepath.Join(repoDir, ".gitignore")
	giBefore, err := os.ReadFile(giPath) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read .gitignore after WriteMCP: %v", err)
	}
	if !strings.Contains(string(giBefore), ".mcp.json") {
		t.Fatalf(".gitignore missing .mcp.json entry after WriteMCP: %q", giBefore)
	}

	if _, err = Remove(repoDir); err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected .mcp.json to survive (user server present), stat err = %v", statErr)
	}
	giAfter, readErr := os.ReadFile(giPath) //nolint:gosec // temp dir.
	if readErr == nil {
		t.Errorf(".gitignore still holds the .mcp.json entry after RemoveMCP: %q", giAfter)
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		t.Fatalf("read .gitignore after RemoveMCP: %v", readErr)
	}
}

func TestRemoveMCP_Unparseable(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	path := filepath.Join(repoDir, ".mcp.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := Remove(repoDir)
	if !errors.Is(err, jsondoc.ErrUnparseableJSON) {
		t.Fatalf("err = %v, want fsx.ErrUnparseableJSON", err)
	}
	got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != original {
		t.Errorf("file was rewritten: got %q, want unchanged %q", got, original)
	}
}

func TestRemoveMCP_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	changed, err := Remove(t.TempDir())
	if err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}
	if changed {
		t.Error("changed = true, want false (nothing was there to remove)")
	}
}

func TestRemoveMCP_ChangedSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		wantChanged bool
	}{
		{name: "absent file", wantChanged: false},
		{
			name:        "present with no gopls entry",
			seed:        `{"mcpServers": {"other": {"command": "other"}}}`,
			wantChanged: false,
		},
		{
			name:        "present with gopls entry",
			seed:        `{"mcpServers": {"gopls": {"command": "gopls", "args": ["mcp"]}}}`,
			wantChanged: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			repoDir := t.TempDir()
			if testCase.seed != "" {
				path := filepath.Join(repoDir, ".mcp.json")
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			changed, err := Remove(repoDir)
			if err != nil {
				t.Fatalf("RemoveMCP: %v", err)
			}
			if changed != testCase.wantChanged {
				t.Errorf("changed = %v, want %v", changed, testCase.wantChanged)
			}
		})
	}
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestRemoveMCP_Idempotent(t *testing.T) {
	fakeOnPath(t, "gopls")

	repoDir := t.TempDir()
	if _, _, err := Write(
		repoDir,
		detect.NewSet(language.Go),
	); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	if _, err := Remove(repoDir); err != nil {
		t.Fatalf("RemoveMCP #1: %v", err)
	}
	if _, err := Remove(repoDir); err != nil {
		t.Fatalf("RemoveMCP #2: %v", err)
	}
}
