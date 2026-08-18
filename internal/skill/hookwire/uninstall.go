package hookwire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// UninstallHook removes the hook script and README InstallHook copied into
// <claudeDir>/hooks, and the installer binary InstallHook copied into
// <claudeDir>/bin, then removes those two directories once they are left
// empty. It is idempotent: a file InstallHook never wrote is simply already
// absent, so re-running is a no-op, not an error. Removing an asset that
// uncovers a backup InstallHook left behind (see marker.Backup) restores
// that backup to the asset's path, completing the reversal; restored reports
// each such restore as a ready-to-print message.
func UninstallHook(claudeDir string) (restored []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	for _, asset := range hookAssets {
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
			return restored, fmt.Errorf("%w", restoreErr)
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

// removeIfInstalled deletes path only when its content carries
// marker.Installed, so a same-named file the user wrote survives untouched.
// It reports whether it removed the file.
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
