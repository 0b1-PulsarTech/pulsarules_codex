package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

const gitignoreName = ".gitignore"

func writeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(content), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", filePath, err)
	}
	return nil
}

func backupIfPresent(filePath string) (backedUp string, err error) {
	if _, statErr := os.Lstat(filePath); statErr != nil {
		return "", nil //nolint:nilerr // absent is the expected "nothing to back up" case.
	}
	backupPath, backupErr := marker.Backup(filePath)
	if backupErr != nil {
		return "", fmt.Errorf("replace existing file: %w", backupErr)
	}
	return marker.BackupMessage(filePath, backupPath), nil
}

func WriteDoc(dir, docName, body string) (backedUp []string, err error) {
	docPath := filepath.Join(dir, docName)
	gitignorePath := filepath.Join(dir, gitignoreName)
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
	// The ignore lands first: writing the doc first and failing here
	// would leave untracked output on disk; reversed, a failure leaves at
	// worst an orphaned .gitignore that ignores itself.
	if err = gitignore.Ensure(dir, docName, gitignoreName); err != nil {
		return backedUp, fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	if err = writeFile(docPath, body); err != nil {
		return backedUp, fmt.Errorf("write %q: %w", docName, err)
	}
	return backedUp, nil
}
