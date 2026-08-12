package agentswire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// RemoveAgents deletes <projectDir>/AGENTS.md, undoing WriteAgents, but only
// when its content carries marker.Installed: a user-authored AGENTS.md this
// tool never wrote survives untouched. It reports whether it removed the
// file, and is idempotent - a missing file, or one lacking the marker, is
// not an error.
func RemoveAgents(projectDir string) (removed bool, err error) {
	path := filepath.Join(projectDir, "AGENTS.md")
	var exists, ours bool
	exists, ours, err = marker.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	if !exists || !ours {
		return false, nil
	}
	if err = os.Remove(path); err != nil {
		return false, fmt.Errorf("remove %q: %w", path, err)
	}
	return true, nil
}
