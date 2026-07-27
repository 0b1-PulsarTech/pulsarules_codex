package docs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
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

func Write(dir, docName, body string) error {
	if err := writeFile(filepath.Join(dir, docName), body); err != nil {
		return fmt.Errorf("write %q: %w", docName, err)
	}
	gitignoreBody := docName + "\n.gitignore\n"
	if err := writeFile(filepath.Join(dir, ".gitignore"), gitignoreBody); err != nil {
		return fmt.Errorf("gitignore for %q: %w", docName, err)
	}
	return nil
}
