package githook

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const binaryName = "pulsarules_cli"

func Install(dir string, hooks []string, opts Options) (backedUp []string, err error) {
	if len(hooks) == 0 {
		return nil, nil
	}
	dest := hooksDir(dir)
	if err = os.MkdirAll(dest, disk.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir %q: %w", dest, err)
	}
	for _, name := range hooks {
		script, ok := hookScript(name, opts)
		if !ok {
			return backedUp, fmt.Errorf(
				"unknown git hook %q (want %s)", name, strings.Join(HookNames(), "|"),
			)
		}
		path := filepath.Join(dest, name)
		exists, ours, checkErr := ownedfile.Check(path)
		if checkErr != nil {
			return backedUp, fmt.Errorf("check %q: %w", path, checkErr)
		}
		if exists && !ours {
			backupPath, backupErr := ownedfile.Backup(path)
			if backupErr != nil {
				return backedUp, fmt.Errorf("back up %q: %w", path, backupErr)
			}
			backedUp = append(backedUp, ownedfile.BackupMessage(path, backupPath))
		} else {
			// An owned or absent file is removed so mode always applies, even over a
			// read-only prior install; the error adds nothing the write would not say.
			_ = os.Remove(path)
		}
		const hookMode = 0o500
		if writeErr := os.WriteFile(path, []byte(script), hookMode); writeErr != nil {
			return backedUp, fmt.Errorf("write hook %q: %w", name, writeErr)
		}
	}
	return backedUp, nil
}

func InstallBinary(dir string) error {
	dst := filepath.Join(hooksDir(dir), binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

func HookNames() []string {
	return slices.Sorted(maps.Keys(hookSpecs()))
}

func Uninstall(dir string) (removed, restored []string, err error) {
	dest := hooksDir(dir)
	for name := range hookSpecs() {
		path := filepath.Join(dest, name)
		removedOK, note, uninstallErr := ownedfile.UninstallFile(path)
		if uninstallErr != nil {
			return removed, restored, fmt.Errorf("uninstall hook %q: %w", name, uninstallErr)
		}
		if !removedOK {
			continue
		}
		removed = append(removed, name)
		if note != "" {
			restored = append(restored, note)
		}
	}
	if err = os.RemoveAll(filepath.Join(dest, binaryName)); err != nil {
		return removed, restored, fmt.Errorf("remove installer binary: %w", err)
	}
	return removed, restored, nil
}
