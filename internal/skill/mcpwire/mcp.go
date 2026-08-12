package mcpwire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
)

// server is one MCP server the installer wires in. Add another by appending
// to managedServers - the template ranges over them.
type server struct {
	Name    string
	Command string
	Args    []string
}

// managedServers are the MCP servers the installed skills assume.
var managedServers = []server{
	{Name: "gopls", Command: "gopls", Args: []string{"mcp"}},
}

// templateData feeds mcp.json.tmpl: the absolute repo path plus the server list.
type templateData struct {
	RepoDir string
	Servers []server
}

// mcpConfig is the top-level .mcp.json shape; only mcpServers is managed here,
// every other top-level key is preserved through a merge.
type mcpConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// WriteMCP merges the rendered gopls server into <repoDir>/.mcp.json, preserving
// any other servers and unrelated top-level keys. It is idempotent: re-running
// replaces only the managed gopls entry and never duplicates.
func WriteMCP(templates fs.FS, repoDir string) error {
	servers, err := renderServers(templates, repoDir)
	if err != nil {
		return fmt.Errorf("render servers: %w", err)
	}
	path := filepath.Join(repoDir, ".mcp.json")
	existing, err := os.ReadFile(path) //nolint:gosec // path is the caller's project root.
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %q: %w", path, err)
	}
	merged, err := mergeServers(existing, servers)
	if err != nil {
		return fmt.Errorf("merge servers: %w", err)
	}
	if err = fsx.Save(path, merged); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	if err = gitignore.Ensure(repoDir, ".mcp.json"); err != nil {
		return fmt.Errorf("ensure mcp gitignore: %w", err)
	}
	return nil
}

// renderServers executes the MCP template over the managed server list and
// returns the resulting mcpServers map.
func renderServers(templates fs.FS, repoDir string) (map[string]json.RawMessage, error) {
	tmpl, err := template.ParseFS(templates, "mcp/mcp.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse mcp template: %w", err)
	}
	var buf bytes.Buffer
	vars := templateData{RepoDir: repoDir, Servers: managedServers}
	if err = tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("render mcp config (repo dir %q): %w", repoDir, err)
	}
	var cfg mcpConfig
	if err = json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, fmt.Errorf("decode rendered mcp config: %w", err)
	}
	return cfg.MCPServers, nil
}

// mergeServers folds the managed servers into the existing .mcp.json, preserving
// every other top-level key and every unmanaged server. existing is empty for a
// fresh file.
func mergeServers(
	existing []byte,
	servers map[string]json.RawMessage,
) (map[string]json.RawMessage, error) {
	config := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &config); err != nil {
			return nil, fmt.Errorf("parse existing .mcp.json: %w", err)
		}
	}

	merged := map[string]json.RawMessage{}
	if raw, ok := config["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &merged); err != nil {
			return nil, fmt.Errorf("parse existing mcpServers: %w", err)
		}
	}
	maps.Copy(merged, servers)

	rawServers, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal mcpServers: %w", err)
	}
	config["mcpServers"] = rawServers
	return config, nil
}
