package agents

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func RemoveAgents(projectDir string) (removed bool, err error) {
	path := filepath.Join(projectDir, "AGENTS.md")
	var exists, ours bool
	exists, ours, err = ownedfile.Check(path)
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
