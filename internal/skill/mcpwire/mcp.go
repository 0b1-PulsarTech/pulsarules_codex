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

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
)

type server struct {
	Name    string
	Command string
	Args    []string
}

func managedServers() []server {
	return []server{
		{Name: "gopls", Command: "gopls", Args: []string{"mcp"}},
	}
}

type templateData struct {
	RepoDir string
	Servers []server
}

type mcpConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

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
	//nolint:gosec // path is the caller's project root.
	if err = os.WriteFile(path, merged, fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = gitignore.Ensure(repoDir, ".mcp.json"); err != nil {
		return fmt.Errorf("ensure mcp gitignore: %w", err)
	}
	return nil
}

func renderServers(templates fs.FS, repoDir string) (map[string]json.RawMessage, error) {
	tmpl, err := template.ParseFS(templates, "mcp/mcp.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse mcp template: %w", err)
	}
	var buf bytes.Buffer
	vars := templateData{RepoDir: repoDir, Servers: managedServers()}
	if err = tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("render mcp config (repo dir %q): %w", repoDir, err)
	}
	var cfg mcpConfig
	if err = json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, fmt.Errorf("decode rendered mcp config: %w", err)
	}
	return cfg.MCPServers, nil
}

func mergeServers(existing []byte, servers map[string]json.RawMessage) ([]byte, error) {
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

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal .mcp.json: %w", err)
	}
	return append(out, '\n'), nil
}
