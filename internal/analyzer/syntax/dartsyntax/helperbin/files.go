package helperbin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

type source struct {
	files fs.FS
	root  string
}

func (src source) each(visit func(name string, content []byte, isDir bool) error) error {
	sub, err := fs.Sub(src.files, src.root)
	if err != nil {
		return fmt.Errorf("open embedded helper: %w", err)
	}
	err = fs.WalkDir(sub, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return visit(name, nil, true)
		}
		content, readErr := fs.ReadFile(sub, name)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", name, readErr)
		}
		return visit(name, content, false)
	})
	if err != nil {
		return fmt.Errorf("walk helper files: %w", err)
	}
	return nil
}

func (src source) hash() (string, error) {
	digest := sha256.New()
	err := src.each(func(name string, content []byte, isDir bool) error {
		if isDir {
			return nil
		}
		// The root stays in the key so a cache built by an older binary is still found.
		digest.Write([]byte(path.Join(src.root, name)))
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

func (src source) materialise(target string) error {
	err := src.each(func(name string, content []byte, isDir bool) error {
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
