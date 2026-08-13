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
	exists, ours, checkErr := marker.Check(pluginPath)
	if checkErr != nil {
		return nil, fmt.Errorf("check plugin: %w", checkErr)
	}
	if exists && !ours {
		backupPath, backupErr := marker.Backup(pluginPath)
		if backupErr != nil {
			return nil, fmt.Errorf("%w", backupErr)
		}
		backedUp = append(backedUp, marker.BackupMessage(pluginPath, backupPath))
	}
	if err = os.WriteFile(pluginPath, script, fsperm.FilePrivate); err != nil {
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
// gitignore entry Install wrote into <dir>/.opencode/. The plugin is
// removed only when it carries marker.Installed, so a hand-authored
// same-named file survives untouched. It is idempotent, and reports
// whether the plugin was actually ours and removed.
func Uninstall(dir string) (removed bool, err error) {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	pluginPath := filepath.Join(pluginsDir, pluginName)
	_, ours, checkErr := marker.Check(pluginPath)
	if checkErr != nil {
		return false, fmt.Errorf("check plugin: %w", checkErr)
	}
	if ours {
		if err = os.Remove(pluginPath); err != nil {
			return false, fmt.Errorf("remove plugin: %w", err)
		}
		removed = true
	}
	if err = fsx.RemoveEmptyDir(pluginsDir); err != nil {
		return false, fmt.Errorf("remove empty plugins dir: %w", err)
	}
	binDir := filepath.Join(dir, ".opencode", "bin")
	if err = os.RemoveAll(filepath.Join(dir, binaryRel)); err != nil {
		return false, fmt.Errorf("remove installer binary: %w", err)
	}
	if err = fsx.RemoveEmptyDir(binDir); err != nil {
		return false, fmt.Errorf("remove empty bin dir: %w", err)
	}
	if _, err = gitignore.Remove(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return false, fmt.Errorf("remove opencode gitignore entry: %w", err)
	}
	return removed, nil
}
