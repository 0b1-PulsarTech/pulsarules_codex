package marker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
