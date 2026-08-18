package mcpwire

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
)

// RemoveMCP removes the managed servers (gopls) from <repoDir>/.mcp.json,
// undoing WriteMCP. It drops the mcpServers key once empty, and deletes the
// file (plus the ".mcp.json" gitignore entry) once nothing else remains.
// Invalid JSON leaves the file untouched (wraps fsx.ErrUnparseableJSON);
// an already-absent file, key, or server makes re-running a no-op.
func RemoveMCP(repoDir string) error {
	path := filepath.Join(repoDir, ".mcp.json")
	existing, err := os.ReadFile(path) //nolint:gosec // path is the caller's project root.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %q: %w", path, err)
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return fmt.Errorf("%w: %q: %w", fsx.ErrUnparseableJSON, path, err)
	}

	changed, err := fsx.StripMapSection(config, "mcpServers", fmt.Sprintf("%q mcpServers", path),
		removeManagedServers,
	)
	if err != nil {
		return fmt.Errorf("strip mcpServers: %w", err)
	}
	if !changed {
		return nil
	}
	// why: the gitignore entry is stripped whenever mcpServers changed, not
	// only when the whole file was deleted - a surviving .mcp.json (the
	// user's own mcpServers entries kept it alive) would otherwise leave the
	// entry and marker comment orphaned in .gitignore forever.
	if _, err = fsx.SaveOrRemove(path, config); err != nil {
		return fmt.Errorf("write mcp config: %w", err)
	}
	if _, err = gitignore.Remove(repoDir, ".mcp.json"); err != nil {
		return fmt.Errorf("remove mcp gitignore entry: %w", err)
	}
	return nil
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
