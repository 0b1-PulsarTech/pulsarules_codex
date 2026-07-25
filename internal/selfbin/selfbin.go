package selfbin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func Copy(dst string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve running binary: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(dst), disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", filepath.Dir(dst), err)
	}
	exeBytes, err := os.ReadFile(exe) //nolint:gosec // copying our own executable.
	if err != nil {
		return fmt.Errorf("read running binary: %w", err)
	}
	//nolint:gosec // must be runnable; dst is the caller-chosen hook/plugin path.
	if err = os.WriteFile(dst, exeBytes, disk.FileExec); err != nil {
		return fmt.Errorf("write %q: %w", dst, err)
	}
	return nil
}
