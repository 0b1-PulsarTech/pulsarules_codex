package opencodewire

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// legacyAgentsPath is where WriteAgents used to render AGENTS.md before the
// migration to the project root, relative to the project dir.
const legacyAgentsPath = ".opencode/AGENTS.md"

// legacyAgentsFingerprint is the opencode-specific sentence the
// pre-migration AGENTS.md.tmpl rendered, carrying no project-specific data.
// RetireLegacyAgents uses it as proof of ownership because the legacy file
// predates marker.Installed, so a marker check cannot tell it from a user's
// own file; the sentence is specific enough a hand-authored file won't match.
const legacyAgentsFingerprint = "opencode has no SessionStart hook, so this contract is stated here instead"

// RetireLegacyAgents removes <projectDir>/.opencode/AGENTS.md when its
// content carries legacyAgentsFingerprint, proof it is a pre-migration
// render and not user-authored, and reports whether it did. A file that
// fails the fingerprint check is left in place, with warning explaining
// why; a missing file is not an error, so an already-migrated project is untouched.
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
