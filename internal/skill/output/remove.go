package output

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

type RemoveResult struct {
	Removed  []string
	Restored []string
	Orphaned []string
}

func RemoveDocs(dest, docName string) (RemoveResult, error) {
	var result RemoveResult
	entries, err := os.ReadDir(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}
		return result, fmt.Errorf("read dir %q: %w", dest, err)
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
		dirRestored, dirOrphaned, rmErr := removeOwnedDoc(dir, docName)
		if rmErr != nil {
			return result, rmErr
		}
		result.Removed = append(result.Removed, id)
		result.Restored = append(result.Restored, dirRestored...)
		result.Orphaned = append(result.Orphaned, dirOrphaned...)
	}

	if err = fsx.RemoveEmptyDir(dest); err != nil {
		return result, fmt.Errorf("remove empty dir: %w", err)
	}
	return result, nil
}

func removeOwnedDoc(dir, docName string) (restored, orphaned []string, err error) {
	docPath := filepath.Join(dir, docName)
	if err = os.Remove(docPath); err != nil {
		return nil, nil, fmt.Errorf("remove %q: %w", docPath, err)
	}
	if _, err = gitignore.Remove(dir, docName, gitignoreName); err != nil {
		return nil, nil, fmt.Errorf("remove gitignore entries in %q: %w", dir, err)
	}
	gitignorePath := filepath.Join(dir, gitignoreName)
	// Restore skips a path gitignore.Remove left standing, so a surviving
	// user gitignore is never clobbered by a backup.
	for _, path := range []string{docPath, gitignorePath} {
		if _, statErr := os.Lstat(path); statErr == nil {
			continue
		}
		restoredOK, restoreErr := marker.Restore(path)
		if restoreErr != nil {
			return restored, orphaned, fmt.Errorf("restore backup: %w", restoreErr)
		}
		if restoredOK {
			restored = append(restored, marker.RestoreMessage(path))
		}
	}
	// Queried before the dir goes: a numbered slot survives Restore.
	if orphaned, err = marker.OrphanNotes(docPath, gitignorePath); err != nil {
		return restored, orphaned, err
	}
	if err = fsx.RemoveEmptyDir(dir); err != nil {
		return restored, orphaned, fmt.Errorf("remove empty dir: %w", err)
	}
	return restored, orphaned, nil
}

func isOwnedDoc(dir, docName string) bool {
	_, ours, err := marker.Check(filepath.Join(dir, docName))
	return err == nil && ours
}
