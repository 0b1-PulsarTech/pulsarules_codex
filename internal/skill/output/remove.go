package output

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// RemoveDocs deletes the <docName> and sibling .gitignore WriteDoc wrote
// under each subdirectory of dest, then dest once empty, restoring any
// backup the removal uncovers (reported in restored).
// why: a user's own file, or a references/ dir the skill format ships,
// must survive; a directory we do not recognise is not ours to touch.
func RemoveDocs(dest, docName string) (removed, restored []string, err error) {
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

	if err = fsx.RemoveEmptyDir(dest); err != nil {
		return removed, restored, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, restored, nil
}

// removeOwnedDoc deletes exactly docName and .gitignore from dir, then dir
// itself once empty, so anything else (a user's file, or a
// references/scripts/assets/ subdirectory the skill format ships) survives.
// The .gitignore may already be gone (WriteDoc invites deleting it to
// commit the doc), so removal tolerates fs.ErrNotExist; restored reports any backup.
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
		restoredOK, restoreErr := marker.Restore(path)
		if restoreErr != nil {
			return restored, fmt.Errorf("%w", restoreErr)
		}
		if restoredOK {
			restored = append(restored, marker.RestoreMessage(path))
		}
	}
	if err = fsx.RemoveEmptyDir(dir); err != nil {
		return restored, fmt.Errorf("remove empty dir: %w", err)
	}
	return restored, nil
}

// isOwnedDoc reports whether dir already holds a docName this tool
// installed: either docName carries marker.Installed, or - for a doc
// rendered before that marker existed - the sibling .gitignore matches
// WriteDoc's fingerprint. Either proof suffices, so a pre-marker doc still
// removes cleanly, even with its .gitignore deleted to commit it.
func isOwnedDoc(dir, docName string) bool {
	docPath := filepath.Join(dir, docName)
	if _, ours, err := marker.Check(docPath); err == nil && ours {
		return true
	}
	body, err := os.ReadFile(filepath.Join(dir, ".gitignore")) //nolint:gosec // dir is under dest.
	if err != nil || string(body) != docName+"\n.gitignore\n" {
		return false
	}
	_, err = os.Stat(docPath)
	return err == nil
}
