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
