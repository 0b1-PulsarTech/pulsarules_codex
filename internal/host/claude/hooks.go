package claude

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
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

func InstallHook(templates fs.FS, claudeDir string) error {
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", hooksDir, err)
	}
	for _, asset := range hookAssets() {
		assetBytes, readErr := fs.ReadFile(templates, "hooks/"+asset.name)
		if readErr != nil {
			return fmt.Errorf("read template hooks/%s: %w", asset.name, readErr)
		}
		dst := filepath.Join(hooksDir, asset.name)
		if writeErr := os.WriteFile(dst, assetBytes, asset.mode); writeErr != nil {
			return fmt.Errorf("write %q: %w", dst, writeErr)
		}
	}
	return installBinary(claudeDir)
}

func installBinary(claudeDir string) error {
	dst := filepath.Join(claudeDir, "bin", binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}
