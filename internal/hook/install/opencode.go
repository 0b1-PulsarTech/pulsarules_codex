package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(ctx Context) error {
	if err := opencodehook.Install(ctx.Dir, ctx.Templates); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}

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
