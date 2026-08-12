package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(ctx Context) error {
	if err := opencode.Install(ctx.Dir, ctx.Templates); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}

func (opencodeInstaller) Uninstall(ctx UninstallContext) (Result, error) {
	removed, err := opencode.Uninstall(ctx.Dir)
	if err != nil {
		return Result{}, fmt.Errorf("uninstall opencode hook: %w", err)
	}
	var result Result
	if removed {
		result.Removed = []string{"plugin"}
	}
	return result, nil
}
