package docs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func writeFile(filePath, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filePath, []byte(content), disk.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", filePath, err)
	}
	return nil
}

func backupIfPresent(filePath string) (backedUp string, err error) {
	if _, statErr := os.Lstat(filePath); statErr != nil {
		return "", nil //nolint:nilerr // absent is the expected "nothing to back up" case.
	}
	backupPath, backupErr := ownedfile.Backup(filePath)
	if backupErr != nil {
		return "", fmt.Errorf("replace existing file: %w", backupErr)
	}
	return ownedfile.BackupMessage(filePath, backupPath), nil
}

func Write(dir, docName, body string) (backedUp []string, err error) {
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
	// The ignore lands first: writing the doc first and failing here
	// would leave untracked output on disk; reversed, a failure leaves at
	// worst an orphaned .gitignore that ignores itself.
	gitignoreBody := docName + "\n.gitignore\n"
	if err = writeFile(gitignorePath, gitignoreBody); err != nil {
		return backedUp, fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	if err = writeFile(docPath, body); err != nil {
		return backedUp, fmt.Errorf("write %q: %w", docName, err)
	}
	return backedUp, nil
}
