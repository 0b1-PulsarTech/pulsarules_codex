package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

func Remove(repoDir string) error {
	path := filepath.Join(repoDir, ".mcp.json")
	existing, err := disk.ReadOptional(path)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return fmt.Errorf("%w: %q: %w", jsondoc.ErrUnparseableJSON, path, err)
	}

	changed, err := jsondoc.StripMapSection(
		config,
		"mcpServers",
		fmt.Sprintf("%q mcpServers", path),
		removeManagedServers,
	)
	if err != nil {
		return fmt.Errorf("strip mcpServers: %w", err)
	}
	if !changed {
		return nil
	}
	// The gitignore entry goes whenever mcpServers changed, or a surviving .mcp.json
	// would leave it orphaned.
	if _, err = jsondoc.SaveOrRemove(path, config); err != nil {
		return fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return nil
}

func removeManagedServers(servers map[string]json.RawMessage) bool {
	changed := false
	for _, srv := range managedServers() {
		if _, present := servers[srv.Name]; present {
			delete(servers, srv.Name)
			changed = true
		}
	}
	return changed
}
