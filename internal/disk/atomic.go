package disk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func WriteFileAtomic(path string, body []byte, mode fs.FileMode) error {
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
