package install

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
)

type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(ctx Context) error {
	if err := githook.Install(ctx.Dir, ctx.GitHooks); err != nil {
		return fmt.Errorf("install git hooks: %w", err)
	}
	return nil
}
