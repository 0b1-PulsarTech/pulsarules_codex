package selfbin

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func Copy(dst string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve running binary: %w", err)
	}
	if sameFile(exe, dst) {
		return nil
	}
	if err = os.MkdirAll(filepath.Dir(dst), disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", filepath.Dir(dst), err)
	}
	exeBytes, err := os.ReadFile(exe) //nolint:gosec // copying our own executable.
	if err != nil {
		return fmt.Errorf("read running binary: %w", err)
	}
	target, err := linkTarget(dst)
	if err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	// Rename replaces the inode, so a running dst is never truncated (ETXTBSY).
	if err = disk.WriteFileAtomic(target, exeBytes, disk.FileExec); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	return nil
}

func linkTarget(dst string) (string, error) {
	stat, err := os.Lstat(dst)
	if errors.Is(err, fs.ErrNotExist) {
		return dst, nil
	}
	if err != nil {
		return "", fmt.Errorf("lstat %q: %w", dst, err)
	}
	if stat.Mode()&os.ModeSymlink == 0 {
		return dst, nil
	}
	target, err := filepath.EvalSymlinks(dst)
	if err != nil {
		return "", fmt.Errorf("resolve symlink %q: %w", dst, err)
	}
	return target, nil
}

func sameFile(src, dst string) bool {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return false
	}
	return os.SameFile(srcInfo, dstInfo)
}
