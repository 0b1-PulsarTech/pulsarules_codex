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
// copies the running binary into <dir>/.opencode/bin/. The plugin hooks into
// experimental.chat.system.transform and tool.execute.after, forwarding each
// to the installer binary's hook command; tool.execute.before is a real
// opencode hook too but is deliberately left unregistered since its output
// has no field the model reads back (see the plugin script's simplification
// comment). It also ensures .opencode/.gitignore ignores the binary
// directory.
func Install(dir string, templates fs.FS) error {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir plugins: %w", err)
	}
	script, err := fs.ReadFile(templates, pluginTemplate)
	if err != nil {
		return fmt.Errorf("read plugin template: %w", err)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	if err = os.WriteFile(pluginPath, script, fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write plugin: %w", err)
	}
	if err = InstallBinary(dir); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if err = gitignore.Ensure(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return fmt.Errorf("ensure opencode gitignore: %w", err)
	}
	return nil
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

// Uninstall removes the governance plugin, the installer binary, and the
// "/bin/" gitignore entry Install wrote into <dir>/.opencode/, undoing
// Install. The plugin file is removed only when its content carries
// marker.Installed, so a same-named file a user hand-authored survives
// untouched. It removes the plugins and bin directories once they are left
// empty, and is idempotent: files Install never wrote are simply already
// absent, so re-running is a no-op, not an error. It reports whether the
// plugin file was actually ours and removed, so a caller can tell a real
// removal from a no-op against a project Install never touched.
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
