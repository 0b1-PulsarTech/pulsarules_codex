package marker

import (
	"fmt"
	"io/fs"
	"os"
)

// InstallFile writes body to path as an owned file. A foreign occupant (it
// exists and carries no marker) is renamed to a backup slot first; an owned
// or absent one is removed so mode always applies fresh - a previous install
// may have left the file read-only (0o500), which os.WriteFile cannot
// truncate through. note is the backup message, "" when nothing was backed up.
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
		// why: path is inside a tree this installer maintains, and a missing
		// file makes this a no-op, so the error adds nothing the write below
		// would not also say.
		_ = os.Remove(path)
	}
	if err = os.WriteFile(path, body, mode); err != nil {
		return note, fmt.Errorf("write %q: %w", path, err)
	}
	return note, nil
}

// UninstallFile removes path when it is ours and restores the base backup slot
// its removal uncovers; a foreign or absent file is left untouched. note is
// the ready-to-print restore message, "" when no backup was there.
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
