package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

type serverEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

func Write(
	repoDir string, languages detect.Set,
) (wired, skipped []string, err error) {
	var available []mcpserver.Server
	available, skipped = mcpserver.Available(languages.Sorted())
	if len(available) == 0 {
		return nil, skipped, nil
	}

	var rendered map[string]json.RawMessage
	rendered, err = renderServers(repoDir, available)
	if err != nil {
		return nil, skipped, fmt.Errorf("render servers: %w", err)
	}
	path := filepath.Join(repoDir, ".mcp.json")
	var doc *jsondoc.Document
	doc, err = jsondoc.Load(path)
	if err != nil {
		return nil, skipped, jsondoc.PrefixParse(err, "merge servers: parse existing .mcp.json")
	}
	if err = mergeServers(doc, rendered); err != nil {
		return nil, skipped, fmt.Errorf("merge servers: %w", err)
	}
	if err = doc.Save(); err != nil {
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

func renderServers(
	repoDir string, servers []mcpserver.Server,
) (map[string]json.RawMessage, error) {
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

func mergeServers(doc *jsondoc.Document, servers map[string]json.RawMessage) error {
	merged, err := jsondoc.Section[map[string]json.RawMessage](doc, "mcpServers")
	if err != nil {
		return jsondoc.PrefixParse(err, "parse existing mcpServers")
	}
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	maps.Copy(merged, servers)
	return jsondoc.SetSection(doc, "mcpServers", merged)
}
