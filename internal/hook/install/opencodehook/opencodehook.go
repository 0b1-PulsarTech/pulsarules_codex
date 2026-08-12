package opencodehook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const binaryRel = ".opencode/bin/pulsarules_cli"

const pluginName = "pulsarules-governance.js"

const pluginTemplate = "hooks/opencode-plugin.js"

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

func InstallBinary(dir string) error {
	// The embedded plugin script hardcodes this path; keep them in sync.
	dst := filepath.Join(dir, binaryRel)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}
