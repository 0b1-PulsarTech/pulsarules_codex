package opencodehook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const binaryRel = ".opencode/bin/pulsarules_cli"

const pluginName = "pulsarules-governance.js"

const pluginTemplate = "hooks/opencode-plugin.js"

func Install(dir string, templates fs.FS) (backedUp []string, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err = os.MkdirAll(pluginsDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir plugins: %w", err)
	}
	script, readErr := fs.ReadFile(templates, pluginTemplate)
	if readErr != nil {
		return nil, fmt.Errorf("read plugin template: %w", readErr)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	note, installErr := marker.InstallFile(pluginPath, script, fsperm.FilePrivate)
	if installErr != nil {
		return backedUp, fmt.Errorf("install plugin: %w", installErr)
	}
	if note != "" {
		backedUp = append(backedUp, note)
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

type UninstallResult struct {
	Removed  bool
	Restored []string
	Orphaned []string
}

func Uninstall(dir string) (UninstallResult, error) {
	var res UninstallResult
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	pluginPath := filepath.Join(pluginsDir, pluginName)
	removed, note, err := marker.UninstallFile(pluginPath)
	if err != nil {
		return res, fmt.Errorf("uninstall plugin: %w", err)
	}
	res.Removed = removed
	// Queried before the dir goes: a numbered slot survives Restore.
	if res.Orphaned, err = marker.OrphanNotes(pluginPath); err != nil {
		return res, err
	}
	if note != "" {
		res.Restored = append(res.Restored, note)
	}
	if err = fsx.RemoveEmptyDir(pluginsDir); err != nil {
		return res, fmt.Errorf("remove empty plugins dir: %w", err)
	}
	binDir := filepath.Join(dir, ".opencode", "bin")
	if err = os.RemoveAll(filepath.Join(dir, binaryRel)); err != nil {
		return res, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = fsx.RemoveEmptyDir(binDir); err != nil {
		return res, fmt.Errorf("remove empty bin dir: %w", err)
	}
	if _, err = gitignore.Remove(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return res, fmt.Errorf("remove opencode gitignore entry: %w", err)
	}
	return res, nil
}
