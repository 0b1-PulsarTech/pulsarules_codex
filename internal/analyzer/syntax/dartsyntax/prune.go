package dartsyntax

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func pruneCache(dir string) error {
	root := filepath.Dir(dir)
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("list helper cache: %w", err)
	}
	var failures []error
	for _, entry := range entries {
		if removable(root, entry, filepath.Base(dir)) {
			if removeErr := os.RemoveAll(filepath.Join(root, entry.Name())); removeErr != nil {
				failures = append(failures, removeErr)
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("prune helper cache: %w", errors.Join(failures...))
	}
	return nil
}

func removable(root string, entry fs.DirEntry, current string) bool {
	if entry.IsDir() {
		return stale(root, entry, current)
	}
	name := entry.Name()
	if !strings.HasPrefix(name, "failed-") || !strings.HasSuffix(name, ".json") {
		return false
	}
	info, err := entry.Info()
	return err == nil && time.Since(info.ModTime()) > failureWindow
}

func stale(root string, entry fs.DirEntry, current string) bool {
	const unusedAfter = 7 * 24 * time.Hour
	name := entry.Name()
	switch {
	case name == current:
		return false
	case strings.HasPrefix(name, "build-"):
		info, err := entry.Info()
		return err == nil && time.Since(info.ModTime()) > BuildTimeout
	case !regexp.MustCompile(`-[0-9a-f]{16}$`).MatchString(name):
		return false
	}
	return time.Since(lastUse(root, entry)) > unusedAfter
}

func lastUse(root string, entry fs.DirEntry) time.Time {
	if stat, err := os.Stat(filepath.Join(root, entry.Name(), exeName())); err == nil {
		return stat.ModTime()
	}
	if stat, err := entry.Info(); err == nil {
		return stat.ModTime()
	}
	return time.Now()
}

func touchExe(dir string) {
	now := time.Now()
	// A failed touch only risks an early prune of an entry that is rebuilt on demand.
	_ = os.Chtimes(filepath.Join(dir, exeName()), now, now)
}
