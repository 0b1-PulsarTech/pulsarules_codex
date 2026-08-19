package marker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const BackupSuffix = ".pulsarules-backup"

func Backup(path string) (backupPath string, err error) {
	backupPath, err = nextBackupSlot(path)
	if err != nil {
		return "", err
	}
	if err = os.Rename(path, backupPath); err != nil {
		return "", fmt.Errorf("backup %q: %w", path, err)
	}
	return backupPath, nil
}

func nextBackupSlot(path string) (string, error) {
	candidate := path + BackupSuffix
	for attempt := 1; ; attempt++ {
		_, statErr := os.Lstat(candidate)
		if errors.Is(statErr, fs.ErrNotExist) {
			return candidate, nil
		}
		if statErr != nil {
			return "", fmt.Errorf("stat %q: %w", candidate, statErr)
		}
		candidate = fmt.Sprintf("%s.%d", path+BackupSuffix, attempt)
	}
}

func Restore(path string) (restored bool, err error) {
	// Restores only the base slot, leaving numbered overflow slots for the user.
	backupPath := path + BackupSuffix
	if _, statErr := os.Lstat(backupPath); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %q: %w", backupPath, statErr)
	}
	if err = os.Rename(backupPath, path); err != nil {
		return false, fmt.Errorf("restore %q: %w", backupPath, err)
	}
	return true, nil
}

func BackupMessage(original, backup string) string {
	return fmt.Sprintf("backed up existing %s to %s", original, backup)
}

func RestoreMessage(path string) string {
	return fmt.Sprintf("restored backup to %s", path)
}

func Orphans(path string) ([]string, error) {
	dir := filepath.Dir(path)
	// Scanned rather than probed slot by slot, so a slot past a numbering gap is found.
	entries, err := os.ReadDir(dir)
	if err != nil {
		// A missing dir is a legitimate never-installed target with no
		// orphans to report, and erroring here would break idempotent uninstall.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan %q: %w", dir, err)
	}
	prefix := filepath.Base(path) + BackupSuffix + "."
	var serials []int
	for _, entry := range entries {
		remainder, found := strings.CutPrefix(entry.Name(), prefix)
		if !found {
			continue
		}
		// The round-trip check rejects a remainder Atoi tolerates but Backup
		// never writes ("007", "+1"), keeping the rebuilt path faithful.
		serial, atoiErr := strconv.Atoi(remainder)
		if atoiErr != nil || strconv.Itoa(serial) != remainder {
			continue
		}
		serials = append(serials, serial)
	}
	slices.Sort(serials)
	slots := make([]string, len(serials))
	for idx, serial := range serials {
		slots[idx] = fmt.Sprintf("%s.%d", path+BackupSuffix, serial)
	}
	return slots, nil
}

func OrphanMessage(path string, slots []string) string {
	return fmt.Sprintf(
		"left %d earlier backup(s) of %s in place: %s",
		len(slots), path, strings.Join(slots, ", "),
	)
}
