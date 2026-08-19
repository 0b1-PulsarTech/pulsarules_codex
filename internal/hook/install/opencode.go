package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
)

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

func (opencodeInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	removed, restored, err := opencodehook.Uninstall(ctx.Dir)
	if err != nil {
		return Result{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	result := Result{Restored: restored}
	if removed {
		result.Removed = []string{"plugin"}
	}
	return result, nil
}
