package disk

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var errSymlink = errors.New("refusing to replace a symlink")

func WriteFileAtomic(path string, body []byte, mode fs.FileMode) error {
	// Rename would swap the link itself for a regular file, severing a link someone placed there.
	if existing, err := os.Lstat(path); err == nil && existing.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %q", errSymlink, path)
	}

	// Staged in the target directory: Rename is atomic only within one filesystem.
	staging, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".staging-*")
	if err != nil {
		return fmt.Errorf("create staging file for %q: %w", path, err)
	}
	stagingPath := staging.Name()
	// Harmless after the rename below succeeds: the staging name is gone by then.
	defer func() { _ = os.Remove(stagingPath) }()

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
