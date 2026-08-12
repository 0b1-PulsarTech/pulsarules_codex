package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func Remove(dest, docName string) (removed, restored []string, err error) {
	var entries []os.DirEntry
	entries, err = os.ReadDir(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read dir %q: %w", dest, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		dir := filepath.Join(dest, id)
		if !isOwnedDoc(dir, docName) {
			continue
		}
		dirRestored, rmErr := removeOwnedDoc(dir, docName)
		if rmErr != nil {
			return removed, restored, rmErr
		}
		removed = append(removed, id)
		restored = append(restored, dirRestored...)
	}

	if err = disk.RemoveEmptyDir(dest); err != nil {
		return removed, restored, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, restored, nil
}

func removeOwnedDoc(dir, docName string) (restored []string, err error) {
	docPath := filepath.Join(dir, docName)
	if err = os.Remove(docPath); err != nil {
		return nil, fmt.Errorf("remove %q: %w", docPath, err)
	}
	gitignorePath := filepath.Join(dir, ".gitignore")
	if rmErr := os.Remove(gitignorePath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("remove %q: %w", gitignorePath, rmErr)
	}
	for _, path := range []string{docPath, gitignorePath} {
		restoredOK, restoreErr := ownedfile.Restore(path)
		if restoreErr != nil {
			return restored, fmt.Errorf("restore backup: %w", restoreErr)
		}
		if restoredOK {
			restored = append(restored, ownedfile.RestoreMessage(path))
		}
	}
	if err = disk.RemoveEmptyDir(dir); err != nil {
		return restored, fmt.Errorf("remove empty dir: %w", err)
	}
	return restored, nil
}

func isOwnedDoc(dir, docName string) bool {
	_, ours, err := ownedfile.Check(filepath.Join(dir, docName))
	return err == nil && ours
}
