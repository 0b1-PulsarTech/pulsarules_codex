package install

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := opencodehook.Install(dir); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}
