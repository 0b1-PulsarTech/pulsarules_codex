package hookwire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func UninstallHook(claudeDir string) (restored []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	for _, asset := range hookAssets() {
		path := filepath.Join(hooksDir, asset.name)
		removedOK, removeErr := removeIfInstalled(path)
		if removeErr != nil {
			return restored, removeErr
		}
		if !removedOK {
			continue
		}
		restoredOK, restoreErr := marker.Restore(path)
		if restoreErr != nil {
			return restored, fmt.Errorf("restore backup of %q: %w", path, restoreErr)
		}
		if restoredOK {
			restored = append(restored, marker.RestoreMessage(path))
		}
	}
	if err = fsx.RemoveEmptyDir(hooksDir); err != nil {
		return restored, fmt.Errorf("remove empty hooks dir: %w", err)
	}
	binDir := filepath.Join(claudeDir, "bin")
	if err = os.RemoveAll(filepath.Join(binDir, binaryName)); err != nil {
		return restored, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = fsx.RemoveEmptyDir(binDir); err != nil {
		return restored, fmt.Errorf("remove empty bin dir: %w", err)
	}
	return restored, nil
}

func removeIfInstalled(path string) (removed bool, err error) {
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
