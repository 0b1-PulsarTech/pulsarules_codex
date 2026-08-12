package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func RemoveEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read dir %q: %w", dir, err)
	}
	if len(entries) > 0 {
		return nil
	}
	if err = os.Remove(dir); err != nil {
		return fmt.Errorf("remove %q: %w", dir, err)
	}
	return nil
}
