package install

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
)

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := opencode.Install(dir); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}
