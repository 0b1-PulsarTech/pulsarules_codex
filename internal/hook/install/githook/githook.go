package githook

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const binaryRel = ".git/hooks/pulsarules_cli"

func Install(dir string, hooks []string) (backedUp []string, err error) {
	if len(hooks) == 0 {
		return nil, nil
	}
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err = os.MkdirAll(hooksDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir .git/hooks: %w", err)
	}
	for _, name := range hooks {
		script, ok := hookScript(name)
		if !ok {
			return backedUp, fmt.Errorf(
				"unknown git hook %q (want %s)", name, strings.Join(HookNames(), "|"),
			)
		}
		path := filepath.Join(hooksDir, name)
		exists, ours, checkErr := marker.Check(path)
		if checkErr != nil {
			return backedUp, fmt.Errorf("check %q: %w", path, checkErr)
		}
		if exists && !ours {
			backupPath, backupErr := marker.Backup(path)
			if backupErr != nil {
				return backedUp, fmt.Errorf("back up %q: %w", path, backupErr)
			}
			backedUp = append(backedUp, marker.BackupMessage(path, backupPath))
		} else {
			// An owned or absent file is removed so mode always applies, even over a
			// read-only prior install; the error adds nothing the write would not say.
			_ = os.Remove(path)
		}
		const hookMode = 0o500
		if writeErr := os.WriteFile(path, []byte(script), hookMode); writeErr != nil {
			return backedUp, fmt.Errorf("write .git/hooks/%s: %w", name, writeErr)
		}
	}
	return backedUp, nil
}

func InstallBinary(dir string) error {
	dst := filepath.Join(dir, ".git", "hooks", "pulsarules_cli")
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

func HookNames() []string {
	return slices.Sorted(maps.Keys(hookSpecs()))
}

func Uninstall(dir string) (removed, restored []string, err error) {
	hooksDir := filepath.Join(dir, ".git", "hooks")
	for name := range hookSpecs() {
		path := filepath.Join(hooksDir, name)
		_, ours, checkErr := marker.Check(path)
		if checkErr != nil {
			return removed, restored, fmt.Errorf("check %q: %w", path, checkErr)
		}
		if !ours {
			continue
		}
		if err = os.Remove(path); err != nil {
			return removed, restored, fmt.Errorf("remove %q: %w", path, err)
		}
		removed = append(removed, name)
		restoredOK, restoreErr := marker.Restore(path)
		if restoreErr != nil {
			return removed, restored, fmt.Errorf("restore %q: %w", path, restoreErr)
		}
		if restoredOK {
			restored = append(restored, marker.RestoreMessage(path))
		}
	}
	if err = os.RemoveAll(filepath.Join(hooksDir, "pulsarules_cli")); err != nil {
		return removed, restored, fmt.Errorf("remove installer binary: %w", err)
	}
	return removed, restored, nil
}
