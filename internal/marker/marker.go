package marker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const Installed = "Installed by pulsarules_cli"

func Check(path string) (exists, ours bool, err error) {
	var content []byte
	content, err = os.ReadFile(path) //nolint:gosec // path is under the caller's directory.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("read %q: %w", path, err)
	}
	return true, strings.Contains(string(content), Installed), nil
}
