package ownedfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
