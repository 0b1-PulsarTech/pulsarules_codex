package ownedfile

import (
	"fmt"
	"io/fs"
	"os"
)

func InstallFile(path string, body []byte, mode fs.FileMode) (note string, err error) {
	var exists, ours bool
	exists, ours, err = Check(path)
	if err != nil {
		return "", fmt.Errorf("check %q: %w", path, err)
	}
	if exists && !ours {
		backupPath, backupErr := Backup(path)
		if backupErr != nil {
			return "", backupErr
		}
		note = BackupMessage(path, backupPath)
	} else {
		// An owned or absent file is removed so mode always applies, even over a
		// read-only prior install; the error adds nothing the write would not say.
		_ = os.Remove(path)
	}
	if err = os.WriteFile(path, body, mode); err != nil {
		return note, fmt.Errorf("write %q: %w", path, err)
	}
	return note, nil
}

func UninstallFile(path string) (removed bool, note string, err error) {
	var ours bool
	_, ours, err = Check(path)
	if err != nil {
		return false, "", fmt.Errorf("check %q: %w", path, err)
	}
	if !ours {
		return false, "", nil
	}
	if err = os.Remove(path); err != nil {
		return false, "", fmt.Errorf("remove %q: %w", path, err)
	}
	restoredOK, restoreErr := Restore(path)
	if restoreErr != nil {
		return true, "", restoreErr
	}
	if restoredOK {
		note = RestoreMessage(path)
	}
	return true, note, nil
}
