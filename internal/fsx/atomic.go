package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrSymlink is returned instead of writing through a symbolic link.
//
// why: a pre-placed link is how a write aimed at a project file lands on an
// arbitrary victim outside it. os.Rename replaces the link rather than
// following it, so the explicit check exists to say so instead of surprising.
var ErrSymlink = errors.New("refusing to write through a symlink")

// WriteFileAtomic writes body to path with mode, via a temp file in path's own
// directory renamed into place, so a crash or a full disk never leaves path
// truncated. Rename is atomic only within one filesystem, hence the same
// directory.
func WriteFileAtomic(path string, body []byte, mode fs.FileMode) error {
	if existing, err := os.Lstat(path); err == nil && existing.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %q", ErrSymlink, path)
	}

	staging, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".staging-*")
	if err != nil {
		return fmt.Errorf("create staging file for %q: %w", path, err)
	}
	stagingPath := staging.Name()
	defer func() { _ = os.Remove(stagingPath) }() // no-op once the rename below succeeds

	if _, err = staging.Write(body); err != nil {
		_ = staging.Close()
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = staging.Close(); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Chmod(stagingPath, mode); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Rename(stagingPath, path); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
