package docs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

type RemoveResult struct {
	Removed []string
	ownedfile.Ledger
}

func Remove(dest, docName string) (RemoveResult, error) {
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
		dirLedger, rmErr := removeOwnedDoc(dir, docName)
		if rmErr != nil {
			return result, rmErr
		}
		result.Removed = append(result.Removed, id)
		result.Restored = append(result.Restored, dirLedger.Restored...)
		result.Orphaned = append(result.Orphaned, dirLedger.Orphaned...)
	}

	if err = disk.RemoveEmptyDir(dest); err != nil {
		return result, fmt.Errorf("remove empty dir: %w", err)
	}
	return result, nil
}

func removeOwnedDoc(dir, docName string) (ownedfile.Ledger, error) {
	docPath := filepath.Join(dir, docName)
	if _, _, err := ownedfile.RemoveOwned(docPath); err != nil {
		return ownedfile.Ledger{}, err
	}
	if _, err := gitignore.Remove(dir, docName, gitignoreName); err != nil {
		return ownedfile.Ledger{}, fmt.Errorf("remove gitignore entries in %q: %w", dir, err)
	}
	// Restore skips a path gitignore.Remove left standing, so a surviving
	// user gitignore is never clobbered by a backup.
	ledger, err := ownedfile.RestoreVacated(docPath, filepath.Join(dir, gitignoreName))
	if err != nil {
		return ledger, err
	}
	if err = disk.RemoveEmptyDir(dir); err != nil {
		return ledger, fmt.Errorf("remove empty dir: %w", err)
	}
	return ledger, nil
}

func isOwnedDoc(dir, docName string) bool {
	_, ours, err := ownedfile.Check(filepath.Join(dir, docName))
	return err == nil && ours
}
