package claude

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

type hookAsset struct {
	name string
	mode os.FileMode
}

func hookAssets() []hookAsset {
	return []hookAsset{
		{"skill-router-reminder.sh", disk.FileExec},
		{"README.md", disk.File},
	}
}

const binaryName = "pulsarules_cli"

func InstallHook(templates fs.FS, claudeDir string) (backedUp []string, err error) {
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err = os.MkdirAll(hooksDir, disk.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets() {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.name)
		if readErr != nil {
			return backedUp, fmt.Errorf("read template hooks/%s: %w", asset.name, readErr)
		}
		dst := filepath.Join(hooksDir, asset.name)
		exists, ours, checkErr := ownedfile.Check(dst)
		if checkErr != nil {
			return backedUp, fmt.Errorf("check %q: %w", dst, checkErr)
		}
		if exists && !ours {
			backupPath, backupErr := ownedfile.Backup(dst)
			if backupErr != nil {
				return backedUp, fmt.Errorf("back up %q: %w", dst, backupErr)
			}
			backedUp = append(backedUp, ownedfile.BackupMessage(dst, backupPath))
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

func installBinary(claudeDir string) error {
	dst := filepath.Join(claudeDir, "bin", binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}
