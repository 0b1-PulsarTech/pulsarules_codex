package marker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Installed is the ownership marker every asset this tool writes into a
// user's project carries, so Check can prove a file predates this tool
// before an installer overwrites it or an uninstaller deletes it.
const Installed = "Installed by pulsarules_cli"

// Check reports whether path exists and, if so, whether its content carries
// Installed. The two bools stay independent - "absent" and "exists but not
// ours" both read as ours == false - so a caller can tell them apart and map
// each to its own decision (skip, warn, overwrite, remove).
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
