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

// BackupSuffix names the sibling file Backup renames a foreign (not-ours)
// file into before an installer overwrites it, so the file survives instead
// of being silently destroyed.
const BackupSuffix = ".pulsarules-backup"

// Backup renames path to the next free "<path>.pulsarules-backup[.N]" slot -
// the bare suffix first, then .1, .2, ... - so it never overwrites a backup
// an earlier install already left behind. It returns the slot path renamed to.
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

// nextBackupSlot finds the first "<path>.pulsarules-backup[.N]" slot that is
// currently free.
func nextBackupSlot(path string) (string, error) {
	candidate := path + BackupSuffix
	for n := 1; ; n++ {
		_, statErr := os.Lstat(candidate)
		if errors.Is(statErr, fs.ErrNotExist) {
			return candidate, nil
		}
		if statErr != nil {
			return "", fmt.Errorf("stat %q: %w", candidate, statErr)
		}
		candidate = fmt.Sprintf("%s.%d", path+BackupSuffix, n)
	}
}

// Restore renames path's backup (path+BackupSuffix) back to path, undoing a
// prior Backup, and reports whether a backup was found. It only restores the
// base slot; a numbered overflow slot (see Backup) is left for the user to
// reconcile, since it is not provably the antecedent of the file being
// uninstalled. Idempotent: a missing backup is not an error.
func Restore(path string) (restored bool, err error) {
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

// Orphans returns the numbered backup slots sitting beside path - the ones
// Restore deliberately leaves behind, since a numbered slot is not provably the
// antecedent of the file being uninstalled. The base slot is excluded: Restore
// consumes that one. The directory is scanned, not probed slot by slot, so a
// slot past a numbering gap (a hand-deleted .1) is still named.
func Orphans(path string) ([]string, error) {
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		// why: a missing dir is a legitimate never-installed target with no
		// orphans to report - erroring here would break idempotent uninstall.
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
	for i, serial := range serials {
		slots[i] = fmt.Sprintf("%s.%d", path+BackupSuffix, serial)
	}
	return slots, nil
}

// OrphanNotes collects one ready-to-print message per path that still has
// numbered backup slots beside it, skipping the paths that have none.
// why: every uninstaller asks the same question about its own set of paths,
// and a leftover slot no caller mentions sits on disk forever.
func OrphanNotes(paths ...string) (notes []string, err error) {
	for _, path := range paths {
		slots, slotsErr := Orphans(path)
		if slotsErr != nil {
			return notes, fmt.Errorf("list backups of %q: %w", path, slotsErr)
		}
		if len(slots) > 0 {
			notes = append(notes, OrphanMessage(path, slots))
		}
	}
	return notes, nil
}

// OrphanMessage names the leftover slots for a person to reconcile by hand.
func OrphanMessage(path string, slots []string) string {
	return fmt.Sprintf(
		"left %d earlier backup(s) of %s in place: %s", len(slots), path, strings.Join(slots, ", "),
	)
}
