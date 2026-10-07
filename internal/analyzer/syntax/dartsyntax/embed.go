package dartsyntax

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

//go:embed helper
var helperFS embed.FS

const helperRoot = "helper"

func helperHash() (string, error) {
	digest := sha256.New()
	err := fs.WalkDir(helperFS, helperRoot, func(
		name string, entry fs.DirEntry, walkErr error,
	) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		content, err := helperFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		digest.Write([]byte(name))
		digest.Write([]byte{0})
		digest.Write(content)
		digest.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("hash embedded helper: %w", err)
	}
	const hashChars = 16
	return hex.EncodeToString(digest.Sum(nil))[:hashChars], nil
}

func materialise(target string) error {
	err := fs.WalkDir(helperFS, helperRoot, func(
		name string, entry fs.DirEntry, walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		relative := path.Clean(name[len(helperRoot):])
		destination := filepath.Join(target, filepath.FromSlash(relative))
		if entry.IsDir() {
			return os.MkdirAll(destination, disk.DirPrivate)
		}
		content, err := helperFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		return os.WriteFile(destination, content, disk.FilePrivate)
	})
	if err != nil {
		return fmt.Errorf("write embedded helper: %w", err)
	}
	return nil
}
