package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

func Remove(repoDir string) (changed bool, err error) {
	path := filepath.Join(repoDir, ".mcp.json")
	var doc *jsondoc.Document
	doc, err = jsondoc.Load(path)
	if err != nil {
		return false, err
	}
	if !doc.Exists() {
		return false, nil
	}

	changed, err = jsondoc.StripMap(doc, "mcpServers", removeManagedServers)
	if err != nil {
		return false, fmt.Errorf("strip mcpServers: %w", err)
	}
	if !changed {
		return false, nil
	}
	// The gitignore entry goes whenever mcpServers changed, or a surviving .mcp.json
	// would leave it orphaned.
	if _, err = doc.SaveOrRemove(); err != nil {
		return false, fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return false, fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return true, nil
}

func removeManagedServers(servers map[string]json.RawMessage) bool {
	changed := false
	// Uninstall cannot know the detected languages, so every managed server is removed.
	for _, name := range ManagedNames() {
		if _, present := servers[name]; present {
			delete(servers, name)
			changed = true
		}
	}
	return changed
}
