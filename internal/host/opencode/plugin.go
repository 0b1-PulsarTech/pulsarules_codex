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
	note, installErr := ownedfile.InstallFile(pluginPath, script, disk.FilePrivate)
	if installErr != nil {
		return backedUp, fmt.Errorf("install plugin: %w", installErr)
	}
	if note != "" {
		backedUp = append(backedUp, note)
	}
	// The embedded plugin script hardcodes this path; keep them in sync.
	if err = selfbin.Install(binDir(dir)); err != nil {
		return backedUp, fmt.Errorf("install binary: %w", err)
	}
	if err = gitignore.Ensure(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return backedUp, fmt.Errorf("ensure opencode gitignore: %w", err)
	}
	return backedUp, nil
}

func binDir(dir string) string {
	return filepath.Join(dir, filepath.Dir(filepath.FromSlash(binaryRel)))
}

type PluginRemoval struct {
	Removed  bool
	Restored []string
	Orphaned []string
}

func Uninstall(dir string) (PluginRemoval, error) {
	var res PluginRemoval
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	pluginPath := filepath.Join(pluginsDir, pluginName)
	removed, note, err := ownedfile.UninstallFile(pluginPath)
	if err != nil {
		return res, fmt.Errorf("uninstall plugin: %w", err)
	}
	res.Removed = removed
	// Queried before the dir goes: a numbered slot survives Restore.
	if res.Orphaned, err = ownedfile.OrphanNotes(pluginPath); err != nil {
		return res, err
	}
	if note != "" {
		res.Restored = append(res.Restored, note)
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
