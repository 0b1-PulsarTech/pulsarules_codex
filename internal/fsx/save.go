package fsx

import (
	"encoding/json"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// Save marshals v as indented JSON, appends a trailing newline, and writes it
// atomically to path with fsperm.FilePrivate.
func Save(path string, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	out = append(out, '\n')

	if err = WriteFileAtomic(path, out, fsperm.FilePrivate); err != nil {
		return err
	}
	return nil
}
