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
