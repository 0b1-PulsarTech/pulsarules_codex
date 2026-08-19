package ownedfile

import (
	"fmt"
	"os"
)

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
