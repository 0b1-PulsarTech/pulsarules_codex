package install

import (
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := githook.Install(dir, []string{"commit-msg", "pre-commit"}); err != nil {
		return fmt.Errorf("install git hooks: %w", err)
	}
	return nil
}
