package codex

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
)

const ConfigFile = "config.toml"

var ErrSymlink = errors.New("is a symbolic link")

func RenderMCP(servers []mcpserver.Server) string { return renderBlock(servers) }

func ClashFound(codexDir, name string) (string, error) {
	text, err := disk.ReadOptional(filepath.Join(codexDir, ConfigFile))
	if err != nil {
		return "", err
	}
	return findClash(string(text), name), nil
}

func refuseSymlink(path string) error {
	// Rewriting would sever the link or write through it; both belong to the user.
	if stat, err := os.Lstat(path); err == nil && stat.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %q", ErrSymlink, path)
	}
	return nil
}

func WriteMCP(codexDir string, servers []mcpserver.Server) (wired, clashed []string, err error) {
	path := filepath.Join(codexDir, ConfigFile)
	if err = refuseSymlink(path); err != nil {
		return nil, nil, err
	}
	var existing []byte
	existing, err = disk.ReadOptional(path)
	if err != nil {
		return nil, nil, err
	}
	var result blockResult
	result, err = applyBlock(string(existing), servers)
	if err != nil {
		return nil, nil, fmt.Errorf("update %q: %w", path, err)
	}
	if result.text == string(existing) {
		return result.wired, result.clashed, nil
	}
	if err = os.MkdirAll(codexDir, disk.ModeDirPrivate); err != nil {
		return nil, nil, fmt.Errorf("mkdir %q: %w", codexDir, err)
	}
	if err = writeConfig(path, result.text); err != nil {
		return nil, nil, err
	}
	return result.wired, result.clashed, nil
}

func RemoveMCP(codexDir string) (changed bool, err error) {
	path := filepath.Join(codexDir, ConfigFile)
	if err = refuseSymlink(path); err != nil {
		return false, err
	}
	var existing []byte
	existing, err = disk.ReadOptional(path)
	if err != nil || existing == nil {
		return false, err
	}
	var result blockResult
	result, err = applyBlock(string(existing), nil)
	if err != nil {
		return false, fmt.Errorf("update %q: %w", path, err)
	}
	if result.text == string(existing) {
		return false, nil
	}
	if strings.TrimSpace(result.text) == "" {
		if err = os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("remove %q: %w", path, err)
		}
		return true, nil
	}
	return true, writeConfig(path, result.text)
}

func writeConfig(path, body string) error {
	mode := disk.ModeFilePrivate
	if stat, err := os.Stat(path); err == nil {
		// The file is the user's: its permissions stay as they set them.
		mode = stat.Mode().Perm()
	}
	if err := disk.WriteFileAtomic(path, []byte(body), mode); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
