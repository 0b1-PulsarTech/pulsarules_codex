package cursorwire

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

const RulesDir = ".cursor/rules"

func WriteRule(projectDir, id, body string) (wrote bool, err error) {
	path := filepath.Join(projectDir, RulesDir, id+".mdc")
	var ours bool
	ours, err = ownsExisting(path)
	if err != nil {
		return false, err
	}
	if !ours {
		return false, nil
	}
	if err = os.MkdirAll(filepath.Dir(path), fsperm.DirPrivate); err != nil {
		return false, fmt.Errorf("mkdir: %w", err)
	}
	// Path is under the caller's project dir.
	if err = os.WriteFile(path, []byte(body), fsperm.FilePrivate); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}
	return true, nil
}

func ownsExisting(path string) (bool, error) {
	exists, ours, err := marker.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	return !exists || ours, nil
}
