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

func Install(dir string, templates fs.FS) (ledger ownedfile.Ledger, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err = os.MkdirAll(pluginsDir, disk.DirPrivate); err != nil {
		return ledger, fmt.Errorf("mkdir plugins: %w", err)
	}
	script, readErr := fs.ReadFile(templates, pluginTemplate)
	if readErr != nil {
		return ledger, fmt.Errorf("read plugin template: %w", readErr)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	note, installErr := ownedfile.InstallFile(pluginPath, script, disk.FilePrivate)
	if installErr != nil {
		return ledger, fmt.Errorf("install plugin: %w", installErr)
	}
	ledger.AddBackup(note)
	// The embedded plugin script hardcodes this path; keep them in sync.
	if err = selfbin.Install(binDir(dir)); err != nil {
		return ledger, fmt.Errorf("install binary: %w", err)
	}
	if err = gitignore.Ensure(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return ledger, fmt.Errorf("ensure opencode gitignore: %w", err)
	}
	return ledger, nil
}

func binDir(dir string) string {
	return filepath.Join(dir, filepath.Dir(filepath.FromSlash(binaryRel)))
}

type PluginRemoval struct {
	Removed bool
	ownedfile.Ledger
}

func Uninstall(dir string) (PluginRemoval, error) {
	var res PluginRemoval
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	removed, ledger, err := ownedfile.UninstallFiles(filepath.Join(pluginsDir, pluginName))
	res.Removed, res.Ledger = len(removed) > 0, ledger
	if err != nil {
		return res, fmt.Errorf("uninstall plugin: %w", err)
	}
	if err = disk.RemoveEmptyDir(pluginsDir); err != nil {
		return res, fmt.Errorf("remove empty plugins dir: %w", err)
	}
	binPath := binDir(dir)
	if err = selfbin.Remove(binPath); err != nil {
		return res, err
	}
	if err = disk.RemoveEmptyDir(binPath); err != nil {
		return res, fmt.Errorf("remove empty bin dir: %w", err)
	}
	if _, err = gitignore.Remove(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return res, fmt.Errorf("remove opencode gitignore entry: %w", err)
	}
	return res, nil
}
