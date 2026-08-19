package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// writeFile creates parent directories and writes content with gosec-safe perms.
func writeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(content), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", filePath, err)
	}
	return nil
}

// backupIfPresent renames an existing file at filePath to a numbered
// ".pulsarules-backup" slot before it is overwritten, reporting the rename
// as a ready-to-print message (empty when nothing was there to back up).
func backupIfPresent(filePath string) (backedUp string, err error) {
	if _, statErr := os.Lstat(filePath); statErr != nil {
		return "", nil //nolint:nilerr // absent is the expected "nothing to back up" case.
	}
	backupPath, backupErr := marker.Backup(filePath)
	if backupErr != nil {
		return "", fmt.Errorf("%w", backupErr)
	}
	return marker.BackupMessage(filePath, backupPath), nil
}

// WriteDoc writes body to <dir>/<docName> plus a sibling .gitignore that
// ignores both, so generated output is untracked by default; delete the
// .gitignore to commit it (docName carries marker.Installed, so ownership
// no longer depends on the .gitignore). A dir isOwnedDoc does not recognize
// is backed up file-by-file first, preserving a user-owned same-named dir.
func WriteDoc(dir, docName, body string) (backedUp []string, err error) {
	docPath := filepath.Join(dir, docName)
	gitignorePath := filepath.Join(dir, ".gitignore")
	if !isOwnedDoc(dir, docName) {
		for _, path := range []string{docPath, gitignorePath} {
			msg, backupErr := backupIfPresent(path)
			if backupErr != nil {
				return backedUp, backupErr
			}
			if msg != "" {
				backedUp = append(backedUp, msg)
			}
		}
	}
	// why: the ignore lands FIRST. Writing the doc first and failing here left
	// generated output on disk with nothing ignoring it - untracked in git,
	// which is the exact state this pair exists to prevent. Reversed, a failure
	// leaves at worst an orphaned .gitignore that ignores itself.
	gitignoreBody := docName + "\n.gitignore\n"
	if err = writeFile(gitignorePath, gitignoreBody); err != nil {
		return backedUp, fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	if err = writeFile(docPath, body); err != nil {
		return backedUp, fmt.Errorf("write %q: %w", docName, err)
	}
	return backedUp, nil
}
