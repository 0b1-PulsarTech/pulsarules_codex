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

// binaryRel is the path to the installer binary relative to the project root,
// placed there by InstallBinary. The embedded plugin script hardcodes this
// same literal since it has no way to read a Go constant; keep them in sync.
const binaryRel = ".opencode/bin/pulsarules_cli"

// pluginName is the file written into .opencode/plugins/.
const pluginName = "pulsarules-governance.js"

// pluginTemplate is the plugin script's path within the embedded templates FS.
const pluginTemplate = "hooks/opencode-plugin.js"

// Install writes the governance plugin into <dir>/.opencode/plugins/ and
// copies the running binary into <dir>/.opencode/bin/. It wires
// experimental.chat.system.transform and tool.execute.after -
// tool.execute.before stays unregistered since its output has no
// model-read field. A pre-existing plugin file is backed up, not destroyed.
func Install(dir string, templates fs.FS) (backedUp []string, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err = os.MkdirAll(pluginsDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir plugins: %w", err)
	}
	var script []byte
	script, err = fs.ReadFile(templates, pluginTemplate)
	if err != nil {
		return nil, fmt.Errorf("read plugin template: %w", err)
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

// InstallBinary copies the running installer binary into <dir>/.opencode/bin/
// so the plugin can invoke it. A copy failure degrades to a no-op plugin.
func InstallBinary(dir string) error {
	dst := filepath.Join(dir, binaryRel)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

// Uninstall removes the governance plugin, installer binary, and "/bin/"
// gitignore entry Install wrote into <dir>/.opencode/. Only a plugin carrying
// marker.Installed is removed - restoring the backup Install left behind
// (reported in restored) - so a hand-authored file survives. Idempotent, and
// reports whether the plugin was actually ours and removed.
func Uninstall(dir string) (removed bool, restored, orphaned []string, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	pluginPath := filepath.Join(pluginsDir, pluginName)
	var note string
	removed, note, err = marker.UninstallFile(pluginPath)
	if err != nil {
		return false, nil, nil, fmt.Errorf("uninstall plugin: %w", err)
	}
	// Queried before the dir goes: a numbered slot survives Restore.
	if orphaned, err = marker.OrphanNotes(pluginPath); err != nil {
		return removed, restored, orphaned, err
	}
	if note != "" {
		restored = append(restored, note)
	}
	if err = fsx.RemoveEmptyDir(pluginsDir); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove empty plugins dir: %w", err)
	}
	binDir := filepath.Join(dir, ".opencode", "bin")
	if err = os.RemoveAll(filepath.Join(dir, binaryRel)); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = fsx.RemoveEmptyDir(binDir); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove empty bin dir: %w", err)
	}
	if _, err = gitignore.Remove(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove opencode gitignore entry: %w", err)
	}
	return removed, restored, orphaned, nil
}
