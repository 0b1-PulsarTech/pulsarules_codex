package disk

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func ReadOptional(path string) ([]byte, error) {
	existing, err := os.ReadFile(path) //nolint:gosec // project-scoped config path
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return existing, nil
}
