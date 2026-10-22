package loader

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func readBounded(path string) ([]byte, error) {
	// One byte past the cap is enough for Parse to reject the file without reading all of it.
	content, err := disk.ReadBounded(path, strictyaml.MaxFileBytes)
	if err != nil {
		return nil, readFault(err)
	}
	return content, nil
}

func readFault(err error) error {
	// The display name already shows the file, and an absolute path could fill the line.
	if pathErr, isPath := errors.AsType[*fs.PathError](err); isPath {
		return fmt.Errorf("read config: %s: %w", pathErr.Op, pathErr.Err)
	}
	return fmt.Errorf("read config: %w", err)
}
