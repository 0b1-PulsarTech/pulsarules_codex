package mcpwire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsonconfig"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
)

type server struct {
	Name    string
	Command string
	Args    []string
}

func managedServersByLanguage() map[langdetect.Language][]server {
	return map[langdetect.Language][]server{
		langdetect.Go:   {{Name: "gopls", Command: "gopls", Args: []string{"mcp"}}},
		langdetect.Dart: {{Name: "dart", Command: "dart", Args: []string{"mcp-server"}}},
	}
}

type templateData struct {
	RepoDir string
	Servers []server
}

type mcpConfig struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

func WriteMCP(
	templates fs.FS, repoDir string, languages langdetect.Set,
) (wired, skipped []string, err error) {
	var available []server
	available, skipped = availableServers(languages)
	if len(available) == 0 {
		return nil, skipped, nil
	}

	var rendered map[string]json.RawMessage
	rendered, err = renderServers(templates, repoDir, available)
	if err != nil {
		return nil, skipped, fmt.Errorf("render servers: %w", err)
	}
	path := filepath.Join(repoDir, ".mcp.json")
	var existing []byte
	existing, err = jsonconfig.Read(path)
	if err != nil {
		return nil, skipped, err
	}
	var merged map[string]json.RawMessage
	merged, err = mergeServers(existing, rendered)
	if err != nil {
		return nil, skipped, fmt.Errorf("merge servers: %w", err)
	}
	if err = fsx.Save(path, merged); err != nil {
		return nil, skipped, fmt.Errorf("save %q: %w", path, err)
	}
	if err = gitignore.Ensure(repoDir, ".mcp.json"); err != nil {
		return nil, skipped, fmt.Errorf("ensure mcp gitignore: %w", err)
	}
	wired = make([]string, len(available))
	for idx, srv := range available {
		wired[idx] = srv.Name
	}
	return wired, skipped, nil
}

func serversFor(languages langdetect.Set) []server {
	var servers []server
	managed := managedServersByLanguage()
	for _, lang := range languages.Sorted() {
		servers = append(servers, managed[lang]...)
	}
	return servers
}

func availableServers(languages langdetect.Set) (available []server, skippedNames []string) {
	for _, srv := range serversFor(languages) {
		if _, lookErr := exec.LookPath(srv.Command); lookErr == nil {
			available = append(available, srv)
		} else {
			skippedNames = append(skippedNames, srv.Name)
		}
	}
	return available, skippedNames
}

func renderServers(
	templates fs.FS, repoDir string, servers []server,
) (map[string]json.RawMessage, error) {
	tmpl, err := template.ParseFS(templates, "mcp/mcp.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse mcp template: %w", err)
	}
	var buf bytes.Buffer
	vars := templateData{RepoDir: repoDir, Servers: servers}
	if err = tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("render mcp config (repo dir %q): %w", repoDir, err)
	}
	var cfg mcpConfig
	if err = json.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, fmt.Errorf("decode rendered mcp config: %w", err)
	}
	return cfg.MCPServers, nil
}

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
