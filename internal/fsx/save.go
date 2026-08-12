package fsx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// Save marshals v as indented JSON, appends a trailing newline, and writes it
// to path with fsperm.FilePrivate. It writes to a temp file in path's own
// directory and renames into place, so a crash or full disk never leaves
// path truncated - a plain os.Rename is atomic only within one filesystem,
// hence the same directory.
func Save(path string, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	out = append(out, '\n')

	staging, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".staging-*")
	if err != nil {
		return fmt.Errorf("create staging file for %q: %w", path, err)
	}
	stagingPath := staging.Name()
	defer func() { _ = os.Remove(stagingPath) }() // no-op once the rename below succeeds

	if _, err = staging.Write(out); err != nil {
		_ = staging.Close()
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = staging.Close(); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Chmod(stagingPath, fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err = os.Rename(stagingPath, path); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
