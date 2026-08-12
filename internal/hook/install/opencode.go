package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(ctx Context) error {
	if err := opencodehook.Install(ctx.Dir); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}
