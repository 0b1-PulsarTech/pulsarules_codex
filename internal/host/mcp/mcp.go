package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

type server struct {
	Name    string
	Command string
	Args    []string
}

func managedServersByLanguage() map[detect.Language][]server {
	return map[detect.Language][]server{
		language.Go:   {{Name: "gopls", Command: "gopls", Args: []string{"mcp"}}},
		language.Dart: {{Name: "dart", Command: "dart", Args: []string{"mcp-server"}}},
	}
}

type serverEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

func Write(
	repoDir string, languages detect.Set,
) (wired, skipped []string, err error) {
	var available []server
	available, skipped = availableServers(languages)
	if len(available) == 0 {
		return nil, skipped, nil
	}

	var rendered map[string]json.RawMessage
	rendered, err = renderServers(repoDir, available)
	if err != nil {
		return nil, skipped, fmt.Errorf("render servers: %w", err)
	}
	path := filepath.Join(repoDir, ".mcp.json")
	var existing []byte
	existing, err = disk.ReadOptional(path)
	if err != nil {
		return nil, skipped, err
	}
	var merged map[string]json.RawMessage
	merged, err = mergeServers(existing, rendered)
	if err != nil {
		return nil, skipped, fmt.Errorf("merge servers: %w", err)
	}
	if err = jsondoc.Save(path, merged); err != nil {
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

func serversFor(languages detect.Set) []server {
	var servers []server
	managed := managedServersByLanguage()
	for _, lang := range languages.Sorted() {
		servers = append(servers, managed[lang]...)
	}
	return servers
}

func availableServers(languages detect.Set) (available []server, skippedNames []string) {
	for _, srv := range serversFor(languages) {
		if _, lookErr := exec.LookPath(srv.Command); lookErr == nil {
			available = append(available, srv)
		} else {
			skippedNames = append(skippedNames, srv.Name)
		}
	}
	return available, skippedNames
}

func renderServers(repoDir string, servers []server) (map[string]json.RawMessage, error) {
	rendered := make(map[string]json.RawMessage, len(servers))
	for _, srv := range servers {
		raw, err := json.Marshal(serverEntry{Command: srv.Command, Args: srv.Args, Cwd: repoDir})
		if err != nil {
			return nil, fmt.Errorf("marshal mcp server %q: %w", srv.Name, err)
		}
		rendered[srv.Name] = raw
	}
	return rendered, nil
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
