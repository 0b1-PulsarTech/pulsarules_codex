package fsx

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveOrRemove persists config to path via Save, or removes path outright
// once config is empty: a config a tool created solely to hold its own
// entries is deleted rather than left behind as an empty file. It reports
// whether it removed path, so a caller with cleanup tied to deletion (e.g.
// a gitignore entry) knows when to run it.
func SaveOrRemove(path string, config map[string]json.RawMessage) (bool, error) {
	if len(config) == 0 {
		if err := os.Remove(path); err != nil {
			return false, fmt.Errorf("remove %q: %w", path, err)
		}
		return true, nil
	}
	if err := Save(path, config); err != nil {
		return false, fmt.Errorf("save %q: %w", path, err)
	}
	return false, nil
}
