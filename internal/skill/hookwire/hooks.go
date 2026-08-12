package hookwire

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

// hookAssets are the files copied verbatim from templates/hooks into
// <claudeDir>/hooks. The script is made executable; the README carries its WHY.
var hookAssets = []struct {
	name string
	mode os.FileMode
}{
	{"skill-router-reminder.sh", fsperm.FileExec},
	{"README.md", fsperm.File},
}

// binaryName is where the orchestrator hook expects the installer binary.
const binaryName = "pulsarules_cli"

// InstallHook copies the hook script (executable) and its README from the
// embedded templates into <claudeDir>/hooks, then installs the binary the
// orchestrator script forwards to. An asset already at one of those paths
// that was NOT written by an earlier InstallHook (no marker.Installed) is
// renamed to a numbered ".pulsarules-backup" slot rather than destroyed;
// backedUp reports each such rename as a ready-to-print message.
func InstallHook(templates fs.FS, claudeDir string) (backedUp []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err = os.MkdirAll(hooksDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.name)
		if readErr != nil {
			return backedUp, fmt.Errorf("read template hooks/%s: %w", asset.name, readErr)
		}
		dst := filepath.Join(hooksDir, asset.name)
		exists, ours, checkErr := marker.Check(dst)
		if checkErr != nil {
			return backedUp, fmt.Errorf("check %q: %w", dst, checkErr)
		}
		if exists && !ours {
			backupPath, backupErr := marker.Backup(dst)
			if backupErr != nil {
				return backedUp, fmt.Errorf("%w", backupErr)
			}
			backedUp = append(backedUp, marker.BackupMessage(dst, backupPath))
		}
		if writeErr := os.WriteFile(dst, assetBytes, asset.mode); writeErr != nil {
			return backedUp, fmt.Errorf("write %q: %w", dst, writeErr)
		}
	}
	if err = installBinary(claudeDir); err != nil {
		return backedUp, err
	}
	return backedUp, nil
}

// installBinary copies the running installer binary into <claudeDir>/bin so the
// orchestrator hook can invoke it. The hook script guards on the binary's
// presence, so a copy failure degrades to a no-op hook rather than a hard error.
func installBinary(claudeDir string) error {
	dst := filepath.Join(claudeDir, "bin", binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}
