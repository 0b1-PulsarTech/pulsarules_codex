package mcpwire

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsonconfig"
)

// RemoveMCP removes the managed servers (gopls) from <repoDir>/.mcp.json,
// undoing WriteMCP. It drops the mcpServers key once empty, and deletes the
// file (plus the ".mcp.json" gitignore entry) once nothing else remains.
// Invalid JSON leaves the file untouched (wraps fsx.ErrUnparseableJSON);
// an already-absent file, key, or server makes re-running a no-op. changed
// reports whether anything on disk actually moved, so a caller can tell a
// real removal from a no-op instead of assuming one from a nil error -
// RemoveMCP returns nil error for both an absent file and a present file
// with no gopls entry.
func RemoveMCP(repoDir string) (changed bool, err error) {
	path := filepath.Join(repoDir, ".mcp.json")
	existing, err := jsonconfig.Read(path)
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
	// why: the gitignore entry is stripped whenever mcpServers changed, not
	// only when the whole file was deleted - a surviving .mcp.json (the
	// user's own mcpServers entries kept it alive) would otherwise leave the
	// entry and marker comment orphaned in .gitignore forever.
	if _, err = fsx.SaveOrRemove(path, config); err != nil {
		return false, fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return false, fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return true, nil
}

// removeManagedServers deletes every managedServers entry from servers,
// reporting whether it removed any.
func removeManagedServers(servers map[string]json.RawMessage) bool {
	changed := false
	for _, srv := range managedServers {
		if _, present := servers[srv.Name]; present {
			delete(servers, srv.Name)
			changed = true
		}
	}
	return changed
}
