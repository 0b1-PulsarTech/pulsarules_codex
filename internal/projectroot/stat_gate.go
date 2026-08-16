package projectroot

import (
	"io/fs"
	"os"
)

const MaxEditedFileBytes = 1 << 20

func StatGate(path string) string {
	stat, err := os.Lstat(path)
	switch {
	case err != nil:
		return "cannot stat"
	case stat.Mode()&fs.ModeSymlink != 0:
		return "is a symlink"
	case !stat.Mode().IsRegular():
		return "not a regular file"
	// This runs on every edit, so a checked-in blob must not stall a turn.
	case stat.Size() > MaxEditedFileBytes:
		return "larger than the size cap"
	}
	return ""
}
