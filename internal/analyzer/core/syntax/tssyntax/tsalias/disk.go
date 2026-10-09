package tsalias

import (
	"io"
	"os"
	"path/filepath"
)

func DiskReader(root string, fallback func(string) ([]byte, bool)) func(string) ([]byte, bool) {
	return func(name string) ([]byte, bool) {
		if root == "" {
			return fallback(name)
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		stat, err := os.Stat(full)
		if err != nil {
			return fallback(name)
		}
		if !stat.Mode().IsRegular() {
			return nil, false
		}
		return readCapped(full)
	}
}

func readCapped(full string) ([]byte, bool) {
	file, err := os.Open(full) //nolint:gosec // The name is a project file already stat-checked.
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()
	body, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil || len(body) > maxConfigBytes {
		return nil, false
	}
	return body, true
}
