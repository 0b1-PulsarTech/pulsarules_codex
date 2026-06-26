package mcpwire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteMCP_FreshFile asserts a .mcp.json is created with the gopls server and
// the repo path substituted into its cwd when none exists.
func TestWriteMCP_FreshFile(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	gopls := readServer(t, repoDir, "gopls")
	if gopls.Command != "gopls" {
		t.Errorf("command = %q, want gopls", gopls.Command)
	}
	if gopls.Cwd != repoDir {
		t.Errorf("cwd = %q, want %q", gopls.Cwd, repoDir)
	}
}

// TestWriteMCP_PreservesExisting asserts re-running preserves unrelated servers
// and top-level keys, sets gopls, and never duplicates.
func TestWriteMCP_PreservesExisting(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	seed := `{
  "someTopLevel": {"keep": true},
  "mcpServers": {
    "other": {"command": "other-mcp", "args": ["serve"], "cwd": "/x"}
  }
}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP #1: %v", err)
	}
	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP #2: %v", err)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var config struct {
		SomeTopLevel json.RawMessage            `json:"someTopLevel"`
		MCPServers   map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(config.SomeTopLevel) == 0 {
		t.Error("lost unrelated top-level key someTopLevel")
	}
	for _, name := range []string{"other", "gopls"} {
		if _, ok := config.MCPServers[name]; !ok {
			t.Errorf("missing server %q", name)
		}
	}
	if len(config.MCPServers) != 2 {
		t.Errorf("server count = %d, want 2 (no duplication)", len(config.MCPServers))
	}
}

// TestWriteMCP_GitignoresMCPJSON asserts .mcp.json (full of machine-specific
// absolute paths) is added to the repo's .gitignore.
func TestWriteMCP_GitignoresMCPJSON(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := WriteMCP(fakeTemplates(), repoDir); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoDir, ".gitignore")) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), ".mcp.json") {
		t.Errorf(".gitignore missing .mcp.json entry, got %q", data)
	}
}

type mcpServer struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
}

func readServer(t testing.TB, repoDir, name string) mcpServer {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoDir, ".mcp.json")) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read .mcp.json: %v", err)
	}
	var config struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}
	server, ok := config.MCPServers[name]
	if !ok {
		t.Fatalf("missing server %q in %+v", name, config.MCPServers)
	}
	return server
}
