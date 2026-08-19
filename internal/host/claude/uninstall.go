package claude

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func UninstallHook(claudeDir string) (restored []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	for _, asset := range hookAssets() {
		path := filepath.Join(hooksDir, asset.destName)
		_, note, removeErr := ownedfile.UninstallFile(path)
		if removeErr != nil {
			return restored, removeErr
		}
		if note != "" {
			restored = append(restored, note)
		}
	}
	if err = disk.RemoveEmptyDir(hooksDir); err != nil {
		return restored, fmt.Errorf("remove empty hooks dir: %w", err)
	}
	binDir := filepath.Join(claudeDir, "bin")
	if err = os.RemoveAll(filepath.Join(binDir, binaryName)); err != nil {
		return restored, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = disk.RemoveEmptyDir(binDir); err != nil {
		return restored, fmt.Errorf("remove empty bin dir: %w", err)
	}
	return restored, nil
}
