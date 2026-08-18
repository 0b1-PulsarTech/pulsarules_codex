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
		return "", nil
	}
	backupPath, backupErr := marker.Backup(filePath)
	if backupErr != nil {
		return "", fmt.Errorf("%w", backupErr)
	}
	return marker.BackupMessage(filePath, backupPath), nil
}

// WriteDoc writes body to <dir>/<docName> plus a sibling .gitignore that
// ignores both docName and itself, so generated skill/workflow output is
// untracked by default; delete that .gitignore to commit the doc to a
// branch (the rendered doc itself carries marker.Installed, so ownership no
// longer depends on the .gitignore surviving that). A dir isOwnedDoc does
// not already recognize as this tool's own is backed up file-by-file (see
// backupIfPresent) before docName and .gitignore are written, so a
// same-named directory a user already owns (e.g. a hand-written "security"
// skill) is preserved rather than destroyed; backedUp reports each such
// rename as a ready-to-print message.
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
	if err = writeFile(docPath, body); err != nil {
		return backedUp, fmt.Errorf("write %q: %w", docName, err)
	}
	gitignoreBody := docName + "\n.gitignore\n"
	if err = writeFile(gitignorePath, gitignoreBody); err != nil {
		return backedUp, fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	return backedUp, nil
}
