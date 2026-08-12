package opencodewire

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const legacyAgentsPath = ".opencode/AGENTS.md"

const legacyAgentsFingerprint = "opencode has no SessionStart hook, " +
	"so this contract is stated here instead"

func RetireLegacyAgents(projectDir string) (removed bool, warning string, err error) {
	path := filepath.Join(projectDir, legacyAgentsPath)
	var content []byte
	//nolint:gosec // path is under the caller's project dir.
	content, err = os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("read %q: %w", path, err)
	}
	// The legacy file predates marker.Installed, so only this sentence proves it is ours.
	if !strings.Contains(string(content), legacyAgentsFingerprint) {
		return false, fmt.Sprintf(
			"found %s but its content does not match the pre-migration template; "+
				"leaving it in place - remove it by hand once you've checked it holds "+
				"nothing you still need", path,
		), nil
	}
	if err = os.Remove(path); err != nil {
		return false, "", fmt.Errorf("remove %q: %w", path, err)
	}
	return true, "", nil
}
