package hookwire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// UninstallHook removes the hook script, README, and installer binary
// InstallHook copied into <claudeDir>/hooks and <claudeDir>/bin, then
// removes those directories once empty. It is idempotent: a file
// InstallHook never wrote is simply absent, a no-op not an error. Removing
// an asset that uncovers a marker.Backup restores it; restored reports each.
func UninstallHook(claudeDir string) (restored []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	for _, asset := range hookAssets {
		path := filepath.Join(hooksDir, asset.destName)
		_, note, removeErr := marker.UninstallFile(path)
		if removeErr != nil {
			return restored, removeErr
		}
		if note != "" {
			restored = append(restored, note)
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
