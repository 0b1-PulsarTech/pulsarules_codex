package mcpwire

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

func TestRemoveMCP_DeletesOurFile(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}

	if err := RemoveMCP(repoDir); err != nil {
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

	if err := RemoveMCP(repoDir); err != nil {
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

func TestRemoveMCP_Unparseable(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	path := filepath.Join(repoDir, ".mcp.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := RemoveMCP(repoDir)
	if !errors.Is(err, fsx.ErrUnparseableJSON) {
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

	if err := RemoveMCP(t.TempDir()); err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}
}

func TestRemoveMCP_Idempotent(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	if err := RemoveMCP(repoDir); err != nil {
		t.Fatalf("RemoveMCP #1: %v", err)
	}
	if err := RemoveMCP(repoDir); err != nil {
		t.Fatalf("RemoveMCP #2: %v", err)
	}
}
