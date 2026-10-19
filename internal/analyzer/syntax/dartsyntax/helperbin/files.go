package helperbin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func eachFile(files fs.FS, visit func(name string, content []byte, isDir bool) error) error {
	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return visit(name, nil, true)
		}
		content, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		return visit(name, content, false)
	})
	if err != nil {
		return fmt.Errorf("walk helper files: %w", err)
	}
	return nil
}

func helperHash(files fs.FS) (string, error) {
	digest := sha256.New()
	err := eachFile(files, func(name string, content []byte, isDir bool) error {
		if isDir {
			return nil
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

func materialise(files fs.FS, target string) error {
	err := eachFile(files, func(name string, content []byte, isDir bool) error {
		destination := filepath.Join(target, filepath.FromSlash(name))
		if isDir {
			return os.MkdirAll(destination, disk.DirPrivate)
		}
		return os.WriteFile(destination, content, disk.FilePrivate)
	})
	if err != nil {
		return fmt.Errorf("write embedded helper: %w", err)
	}
	return nil
}
