package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpserver"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

func Remove(repoDir string) (changed bool, err error) {
	path := filepath.Join(repoDir, ".mcp.json")
	var doc *jsondoc.Document
	doc, err = jsondoc.LoadForRemoval(path)
	if err != nil {
		return false, err
	}
	if !doc.Exists() {
		return false, nil
	}

	changed, err = jsondoc.StripMap(doc, "mcpServers", removeManagedServers(repoDir))
	if err != nil {
		return false, fmt.Errorf("strip mcpServers: %w", err)
	}
	if !changed {
		return false, nil
	}
	// The gitignore entry goes whenever mcpServers changed, or a surviving .mcp.json
	// would leave it orphaned.
	if err = doc.SaveOrRemove(); err != nil {
		return false, fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return false, fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return true, nil
}

func removeManagedServers(repoDir string) func(map[string]json.RawMessage) bool {
	return func(servers map[string]json.RawMessage) bool {
		changed := false
		// Uninstall cannot know the detected languages, so every managed server is removed,
		// but only an entry that is recognisably ours: a user's own server may share the name.
		for _, name := range mcpserver.ManagedNames() {
			if raw, present := servers[name]; present && isOurs(name, raw, repoDir) {
				delete(servers, name)
				changed = true
			}
		}
		return changed
	}
}

func isOurs(name string, raw json.RawMessage, repoDir string) bool {
	var entry serverEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return false
	}
	return mcpserver.IsManaged(name, entry.Command) && sameDir(entry.Cwd, repoDir)
}

func sameDir(first, second string) bool {
	if first == "" || second == "" {
		return false
	}
	if filepath.Clean(first) == filepath.Clean(second) {
		return true
	}
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}
