package jsondoc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func Save(path string, value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	out = append(out, '\n')

	staging, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".staging-*")
	if err != nil {
		return fmt.Errorf("create staging file for %q: %w", path, err)
	}
	stagingPath := staging.Name()
	defer func() { _ = os.Remove(stagingPath) }()

	if _, err = staging.Write(out); err != nil {
		_ = staging.Close()
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = staging.Close(); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Chmod(stagingPath, disk.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Rename(stagingPath, path); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
