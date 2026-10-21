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

func uninstallFile(path string) (removed bool, note string, err error) {
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

func UninstallFiles(
	wrapErr func(path string, err error) error, paths ...string,
) (removed []string, ledger Ledger, err error) {
	// wrapErr words one file's failure as each host always did; an orphan failure stays bare.
	for _, path := range paths {
		removedOK, note, uninstallErr := uninstallFile(path)
		if uninstallErr != nil {
			return removed, ledger, wrapErr(path, uninstallErr)
		}
		if !removedOK {
			continue
		}
		removed = append(removed, path)
		ledger.addRestore(note)
	}
	// Queried before the caller removes the dirs: a numbered slot survives Restore.
	ledger.Orphaned, err = OrphanNotes(paths...)
	return removed, ledger, err
}

func RestoreVacated(paths ...string) (ledger Ledger, err error) {
	for _, path := range paths {
		if _, statErr := os.Lstat(path); statErr == nil {
			continue
		}
		restoredOK, restoreErr := Restore(path)
		if restoreErr != nil {
			return ledger, fmt.Errorf("restore backup: %w", restoreErr)
		}
		if restoredOK {
			ledger.addRestore(RestoreMessage(path))
		}
	}
	// Queried before the caller removes the dir: a numbered slot survives Restore.
	ledger.Orphaned, err = OrphanNotes(paths...)
	return ledger, err
}
