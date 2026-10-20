package pathalias

import (
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func DiskReader(root string, fallback func(string) ([]byte, bool)) func(string) ([]byte, bool) {
	return func(name string) ([]byte, bool) {
		if root == "" {
			return fallback(name)
		}
		full := filepath.Join(root, filepath.FromSlash(name))
		if _, err := os.Stat(full); err != nil {
			return fallback(name)
		}
		body, err := disk.ReadBounded(full, maxConfigBytes)
		if err != nil || len(body) > maxConfigBytes {
			return nil, false
		}
		return body, true
	}
}
