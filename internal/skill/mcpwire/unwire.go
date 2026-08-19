package mcpwire

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsonconfig"
)

func RemoveMCP(repoDir string) (changed bool, err error) {
	path := filepath.Join(repoDir, ".mcp.json")
	var existing []byte
	existing, err = jsonconfig.Read(path)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return false, fmt.Errorf("%w: %q: %w", fsx.ErrUnparseableJSON, path, err)
	}

	changed, err = fsx.StripMapSection(config, "mcpServers", fmt.Sprintf("%q mcpServers", path),
		removeManagedServers,
	)
	if err != nil {
		return false, fmt.Errorf("strip mcpServers: %w", err)
	}
	if !changed {
		return false, nil
	}
	// The gitignore entry goes whenever mcpServers changed, or a surviving .mcp.json
	// would leave it orphaned.
	if _, err = fsx.SaveOrRemove(path, config); err != nil {
		return false, fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return false, fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return true, nil
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
