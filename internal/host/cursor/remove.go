package cursor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

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
		removedOne, ruleErr := removeOwnedRule(filepath.Join(dir, entry.Name()))
		if ruleErr != nil {
			return removed, ruleErr
		}
		if removedOne {
			removed = append(removed, strings.TrimSuffix(entry.Name(), ".mdc"))
		}
	}

	if err = disk.RemoveEmptyDir(dir); err != nil {
		return removed, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, nil
}

func removeOwnedRule(path string) (bool, error) {
	ours, err := ownsForRemoval(path)
	if err != nil || !ours {
		return false, err
	}
	if rmErr := os.Remove(path); rmErr != nil {
		if errors.Is(rmErr, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("remove %q: %w", path, rmErr)
	}
	return true, nil
}

func ownsForRemoval(path string) (bool, error) {
	// Unlike ownsExisting, an absent file is not ours: ReadDir then ReadFile is not atomic.
	_, ours, err := ownedfile.Check(path)
	if err != nil {
		return false, fmt.Errorf("check %q: %w", path, err)
	}
	return ours, nil
}
