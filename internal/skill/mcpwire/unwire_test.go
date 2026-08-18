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

// TestRemoveMCP_DeletesFileWhenOnlyOurs asserts a .mcp.json WriteMCP created
// from scratch is deleted entirely once its managed server is removed, along
// with the gitignore entry WriteMCP added for it.
func TestRemoveMCP_DeletesFileWhenOnlyOurs(t *testing.T) {
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
	// ".mcp.json" was the only entry WriteMCP ever added, so Remove deletes
	// the .gitignore file entirely rather than leave it empty.
	if _, err := os.Stat(filepath.Join(repoDir, ".gitignore")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected .gitignore to be removed once empty, stat err = %v", err)
	}
}

// TestRemoveMCP_PreservesUnrelated asserts an unrelated server and top-level
// key survive, so the file is kept rather than deleted.
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

// TestRemoveMCP_Unparseable asserts a .mcp.json that is not valid JSON is
// left untouched and the error wraps fsx.ErrUnparseableJSON.
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

// TestRemoveMCP_NoOpWhenAbsent asserts a missing .mcp.json is not an error.
func TestRemoveMCP_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	if err := RemoveMCP(t.TempDir()); err != nil {
		t.Fatalf("RemoveMCP: %v", err)
	}
}

// TestRemoveMCP_Idempotent asserts running RemoveMCP twice is not an error.
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
