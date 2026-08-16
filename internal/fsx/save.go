package fsx

import (
	"encoding/json"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

func Save(path string, value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	out = append(out, '\n')

	if err = WriteFileAtomic(path, out, fsperm.FilePrivate); err != nil {
		return err
	}
	return nil
}
