package selfbin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// Copy copies the currently running executable to dst, creating
// dst's parent directory if it does not already exist. The copy is written
// with fsperm.FileExec so it stays directly executable as a hook or plugin
// binary.
func Copy(dst string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve running binary: %w", err)
	}
	if err = os.MkdirAll(filepath.Dir(dst), fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", filepath.Dir(dst), err)
	}
	data, err := os.ReadFile(exe) //nolint:gosec // copying our own executable.
	if err != nil {
		return fmt.Errorf("read running binary: %w", err)
	}
	//nolint:gosec // must be runnable; dst is the caller-chosen hook/plugin path.
	if err = os.WriteFile(dst, data, fsperm.FileExec); err != nil {
		return fmt.Errorf("write %q: %w", dst, err)
	}
	return nil
}
