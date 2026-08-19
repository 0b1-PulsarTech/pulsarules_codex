package jsonconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Read reads path, returning a nil slice (not an error) when the file does
// not exist yet - a fresh config a merge starts empty, or an unmerge treats
// as nothing to unwire. Every wire package's merge and unmerge side shares
// this same "tolerate an absent config file" rule; Read factors it once so
// six call sites stop repeating the fs.ErrNotExist check by hand.
func Read(path string) ([]byte, error) {
	existing, err := os.ReadFile(path) //nolint:gosec // path is the caller's project-scoped config.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return existing, nil
}
