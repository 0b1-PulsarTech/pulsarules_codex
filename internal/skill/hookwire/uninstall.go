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
func UninstallHook(claudeDir string) (restored, orphaned []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	paths := make([]string, 0, len(hookAssets))
	for _, asset := range hookAssets {
		path := filepath.Join(hooksDir, asset.destName)
		paths = append(paths, path)
		_, note, removeErr := marker.UninstallFile(path)
		if removeErr != nil {
			return restored, orphaned, removeErr
		}
		if note != "" {
			restored = append(restored, note)
		}
	}
	// Queried before the dirs go: a numbered slot survives Restore, and
	// nothing else would ever mention it.
	if orphaned, err = marker.OrphanNotes(paths...); err != nil {
		return restored, orphaned, err
	}
	if err = fsx.RemoveEmptyDir(hooksDir); err != nil {
		return restored, orphaned, fmt.Errorf("remove empty hooks dir: %w", err)
	}
	binDir := filepath.Join(claudeDir, "bin")
	if err = os.RemoveAll(filepath.Join(binDir, binaryName)); err != nil {
		return restored, orphaned, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = fsx.RemoveEmptyDir(binDir); err != nil {
		return restored, orphaned, fmt.Errorf("remove empty bin dir: %w", err)
	}
	return restored, orphaned, nil
}
