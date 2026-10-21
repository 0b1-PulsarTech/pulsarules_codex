package claude

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

func bareFailure(_ string, err error) error { return err }

func UninstallHook(claudeDir string) (ledger ownedfile.Ledger, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	assets := hookAssets()
	paths := make([]string, len(assets))
	for idx, asset := range assets {
		paths[idx] = filepath.Join(hooksDir, asset.destName)
	}
	if _, ledger, err = ownedfile.UninstallFiles(bareFailure, paths...); err != nil {
		return ledger, err
	}
	if err = disk.RemoveEmptyDir(hooksDir); err != nil {
		return ledger, fmt.Errorf("remove empty hooks dir: %w", err)
	}
	binDir := filepath.Join(claudeDir, "bin")
	if err = selfbin.Remove(binDir); err != nil {
		return ledger, err
	}
	if err = disk.RemoveEmptyDir(binDir); err != nil {
		return ledger, fmt.Errorf("remove empty bin dir: %w", err)
	}
	return ledger, nil
}
