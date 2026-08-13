package opencode

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

const binaryRel = ".opencode/bin/pulsarules_cli"

const pluginName = "pulsarules-governance.js"

const pluginTemplate = "hooks/opencode-plugin.js"

func Install(dir string, templates fs.FS) (backedUp []string, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err = os.MkdirAll(pluginsDir, disk.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir plugins: %w", err)
	}
	script, readErr := fs.ReadFile(templates, pluginTemplate)
	if readErr != nil {
		return nil, fmt.Errorf("read plugin template: %w", readErr)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	exists, ours, checkErr := ownedfile.Check(pluginPath)
	if checkErr != nil {
		return nil, fmt.Errorf("check plugin: %w", checkErr)
	}
	if exists && !ours {
		backupPath, backupErr := ownedfile.Backup(pluginPath)
		if backupErr != nil {
			return nil, fmt.Errorf("back up %q: %w", pluginPath, backupErr)
		}
		backedUp = append(backedUp, ownedfile.BackupMessage(pluginPath, backupPath))
	}
	if err = os.WriteFile(pluginPath, script, disk.FilePrivate); err != nil {
		return backedUp, fmt.Errorf("write plugin: %w", err)
	}
	if err = InstallBinary(dir); err != nil {
		return backedUp, fmt.Errorf("install binary: %w", err)
	}
	if err = gitignore.Ensure(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return backedUp, fmt.Errorf("ensure opencode gitignore: %w", err)
	}
	return backedUp, nil
}

func InstallBinary(dir string) error {
	// The embedded plugin script hardcodes this path; keep them in sync.
	dst := filepath.Join(dir, binaryRel)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

func Uninstall(dir string) (removed bool, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	pluginPath := filepath.Join(pluginsDir, pluginName)
	_, ours, checkErr := ownedfile.Check(pluginPath)
	if checkErr != nil {
		return false, fmt.Errorf("check plugin: %w", checkErr)
	}
	if ours {
		if err = os.Remove(pluginPath); err != nil {
			return false, fmt.Errorf("remove plugin: %w", err)
		}
		removed = true
	}
	if err = disk.RemoveEmptyDir(pluginsDir); err != nil {
		return false, fmt.Errorf("remove empty plugins dir: %w", err)
	}
	binDir := filepath.Join(dir, ".opencode", "bin")
	if err = os.RemoveAll(filepath.Join(dir, binaryRel)); err != nil {
		return false, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = disk.RemoveEmptyDir(binDir); err != nil {
		return false, fmt.Errorf("remove empty bin dir: %w", err)
	}
	if _, err = gitignore.Remove(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return false, fmt.Errorf("remove opencode gitignore entry: %w", err)
	}
	return removed, nil
}
