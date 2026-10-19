package helperbin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type helperDir string

func exeName() string {
	const exeStem = "summary"
	if runtime.GOOS == "windows" {
		return exeStem + ".exe"
	}
	return exeStem
}

func newHelperDir(base, version, hash string) helperDir {
	return helperDir(filepath.Join(cacheRoot(base), keyPrefix(version)+hash))
}

func cacheRoot(base string) string {
	return filepath.Join(base, "pulsarules", "dart-syntax")
}

func keyPrefix(version string) string {
	return version + "-" + runtime.GOOS + "-" + runtime.GOARCH + "-"
}

func (dir helperDir) parent() string { return filepath.Dir(string(dir)) }

func (dir helperDir) name() string { return filepath.Base(string(dir)) }

func (dir helperDir) exe() string { return filepath.Join(string(dir), exeName()) }

func (dir helperDir) exists() bool {
	_, err := os.Stat(dir.exe())
	return err == nil
}

func (dir helperDir) touch() {
	now := time.Now()
	// A failed touch only risks an early prune of an entry that is rebuilt on demand.
	_ = os.Chtimes(dir.exe(), now, now)
}

func (dir helperDir) publish(work string) error {
	err := os.Rename(work, string(dir))
	if err == nil || dir.exists() {
		return nil
	}
	// A directory without the exe is a crashed or hand-edited leftover.
	if removeErr := os.RemoveAll(string(dir)); removeErr != nil {
		return fmt.Errorf("remove broken helper dir: %w", removeErr)
	}
	if renameErr := os.Rename(work, string(dir)); renameErr != nil && !dir.exists() {
		return fmt.Errorf("publish helper: %w", renameErr)
	}
	return nil
}
