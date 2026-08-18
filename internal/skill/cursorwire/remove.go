package cursorwire

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// RemoveRules deletes every marker-owned .mdc file under
// <projectDir>/.cursor/rules (via ownsForRemoval), then removes the
// directory once empty. A file without the marker is a user's own Cursor
// rule and is left untouched. It reports the removed ids and is idempotent:
// a missing directory, a vanished file, or a second run is not an error.
func RemoveRules(projectDir string) (removed []string, err error) {
	dir := filepath.Join(projectDir, RulesDir)
	var entries []os.DirEntry
	entries, err = os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mdc") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		ours, ownErr := ownsForRemoval(path)
		if ownErr != nil {
			return removed, ownErr
		}
		if !ours {
			continue
		}
		if rmErr := os.Remove(path); rmErr != nil {
			if errors.Is(rmErr, fs.ErrNotExist) {
				continue
			}
			return removed, fmt.Errorf("remove %q: %w", path, rmErr)
		}
		removed = append(removed, strings.TrimSuffix(entry.Name(), ".mdc"))
	}

	if err = fsx.RemoveEmptyDir(dir); err != nil {
		return removed, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, nil
}

// why: unlike ownsExisting, an absent file is "nothing to delete", not
// "ours" - ReadDir followed by ReadFile is not atomic.
func ownsForRemoval(path string) (bool, error) {
	_, ours, err := marker.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	return ours, nil
}
