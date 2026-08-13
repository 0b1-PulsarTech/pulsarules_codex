package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
)

// opencodeInstaller installs the opencode governance plugin + binary +
// .gitignore.
type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(ctx Context) error {
	backedUp, err := opencodehook.Install(ctx.Dir, ctx.Templates)
	if err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	if ctx.Warn != nil {
		for _, msg := range backedUp {
			ctx.Warn("%s", msg)
		}
	}
	return nil
}

// Uninstall removes the governance plugin, binary, and gitignore entry
// Install wrote into ctx.Dir. Result.Removed carries "plugin" only when a
// plugin file actually existed, so a caller can tell a real removal from a
// no-op against a project Install never touched.
func (opencodeInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	removed, err := opencodehook.Uninstall(ctx.Dir)
	if err != nil {
		return Result{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	var result Result
	if removed {
		result.Removed = []string{"plugin"}
	}
	return result, nil
}
