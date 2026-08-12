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
