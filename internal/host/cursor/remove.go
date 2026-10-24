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

func RemoveRules(projectDir string) (removed []string, ledger ownedfile.Ledger, err error) {
	dir := filepath.Join(projectDir, RulesDir)
	var entries []os.DirEntry
	entries, err = os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ledger, nil
		}
		return nil, ledger, fmt.Errorf("read dir %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".mdc") {
			continue
		}
		outcome, saved, ruleErr := ownedfile.RemoveOwned(filepath.Join(dir, entry.Name()))
		ledger.BackedUp = append(ledger.BackedUp, saved.BackedUp...)
		if ruleErr != nil {
			return removed, ledger, ruleErr
		}
		if outcome == ownedfile.Removed {
			removed = append(removed, strings.TrimSuffix(entry.Name(), ".mdc"))
		}
	}

	if err = disk.RemoveEmptyDir(dir); err != nil {
		return removed, ledger, fmt.Errorf("remove empty dir: %w", err)
	}
	return removed, ledger, nil
}
