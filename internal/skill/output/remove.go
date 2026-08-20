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

// RemoveDocs deletes the <docName> and sibling .gitignore WriteDoc wrote
// under each subdirectory of dest, then dest once empty, restoring any
// backup the removal uncovers (reported in restored).
// why: a user's own file, or a references/ dir the skill format ships,
// must survive; a directory we do not recognise is not ours to touch.
func RemoveDocs(dest, docName string) (removed, restored, orphaned []string, err error) {
	var entries []os.DirEntry
	entries, err = os.ReadDir(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("read dir %q: %w", dest, err)
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
			return removed, restored, orphaned, rmErr
		}
		removed = append(removed, id)
		restored = append(restored, dirRestored...)
		orphaned = append(orphaned, dirOrphaned...)
	}

	if err = fsx.RemoveEmptyDir(dest); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, restored, orphaned, nil
}

// removeOwnedDoc deletes exactly docName and its gitignore entries from dir,
// then dir itself once empty, so anything else (a user's file, or a
// references/scripts/assets/ dir the skill format ships) survives. Restore
// skips a path gitignore.Remove left standing (foreign entries kept), so a
// surviving user gitignore is never clobbered by a backup.
func removeOwnedDoc(dir, docName string) (restored, orphaned []string, err error) {
	docPath := filepath.Join(dir, docName)
	if err = os.Remove(docPath); err != nil {
		return nil, nil, fmt.Errorf("remove %q: %w", docPath, err)
	}
	if _, err = gitignore.Remove(dir, docName, gitignoreName); err != nil {
		return nil, nil, fmt.Errorf("remove gitignore entries in %q: %w", dir, err)
	}
	gitignorePath := filepath.Join(dir, gitignoreName)
	for _, path := range []string{docPath, gitignorePath} {
		if _, statErr := os.Lstat(path); statErr == nil {
			continue // still present (foreign content gitignore.Remove kept); nothing to restore.
		}
		restoredOK, restoreErr := marker.Restore(path)
		if restoreErr != nil {
			return restored, orphaned, fmt.Errorf("%w", restoreErr)
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

// isOwnedDoc reports whether dir already holds a docName this tool
// installed, proven the same way every other asset this tool writes proves
// it: docName's own content carries marker.Installed.
func isOwnedDoc(dir, docName string) bool {
	_, ours, err := marker.Check(filepath.Join(dir, docName))
	return err == nil && ours
}
