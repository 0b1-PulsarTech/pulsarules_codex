package jsondoc

import (
	"encoding/json"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func Save(path string, value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}
	out = append(out, '\n')

	if err = disk.WriteFileAtomic(path, out, disk.FilePrivate); err != nil {
		return err
	}
	return nil
}
