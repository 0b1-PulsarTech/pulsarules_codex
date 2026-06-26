package opencodewire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestWireConfig_FreshFile asserts a new opencode.json gets the schema, the
// instruction globs, and the gopls MCP server.
func TestWireConfig_FreshFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}
	config := readConfig(t, projectDir)

	if config.Schema != schemaURL {
		t.Errorf("$schema = %q, want %q", config.Schema, schemaURL)
	}
	for _, glob := range instructionGlobs {
		if !slices.Contains(config.Instructions, glob) {
			t.Errorf("instructions missing %q, got %v", glob, config.Instructions)
		}
	}
	gopls, ok := config.MCP["gopls"]
	if !ok {
		t.Fatalf("missing gopls mcp server, got %v", config.MCP)
	}
	if gopls.Type != "local" || !gopls.Enabled {
		t.Errorf("gopls server = %+v, want type=local enabled=true", gopls)
	}
}

// TestWireConfig_PreservesExisting asserts re-running preserves unrelated keys,
// existing instructions, and other MCP servers, and never duplicates.
func TestWireConfig_PreservesExisting(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "theme": "dark",
  "instructions": ["CONTRIBUTING.md"],
  "mcp": {"other": {"type": "local", "command": ["other"], "enabled": true}}
}`
	path := filepath.Join(projectDir, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig #1: %v", err)
	}
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig #2: %v", err)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := top["theme"]; !ok {
		t.Error("lost unrelated key theme")
	}

	config := readConfig(t, projectDir)
	// Existing + new instructions present, no duplicates.
	want := append([]string{"CONTRIBUTING.md"}, instructionGlobs...)
	if len(config.Instructions) != len(want) {
		t.Errorf("instructions = %v, want %v", config.Instructions, want)
	}
	for _, name := range []string{"other", "gopls"} {
		if _, ok := config.MCP[name]; !ok {
			t.Errorf("missing mcp server %q", name)
		}
	}
}

// TestWireConfig_NoGopls asserts the gopls server is omitted when gopls is absent.
func TestWireConfig_NoGopls(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithoutGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}
	config := readConfig(t, projectDir)
	if _, ok := config.MCP["gopls"]; ok {
		t.Errorf("gopls server present with gopls=WithoutGopls")
	}
}

type opencodeConfig struct {
	Schema       string               `json:"$schema"`
	Instructions []string             `json:"instructions"`
	MCP          map[string]mcpServer `json:"mcp"`
}

type mcpServer struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
}

func readConfig(t testing.TB, projectDir string) opencodeConfig {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(projectDir, "opencode.json")) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read opencode.json: %v", err)
	}
	var config opencodeConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse opencode.json: %v", err)
	}
	return config
}
