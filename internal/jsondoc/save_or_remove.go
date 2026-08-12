package jsondoc

import (
	"encoding/json"
	"fmt"
	"os"
)

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
