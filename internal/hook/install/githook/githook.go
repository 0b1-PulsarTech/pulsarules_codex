package githook

import (
	"fmt"
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
		script, ok := hookScripts()[name]
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

func hookScripts() map[string]string {
	return map[string]string{
		"commit-msg": `#!/bin/sh
# pulsarules_codex commit-msg hook - validates commit message format.
# ` + marker.Installed + `; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" commitlint --project "$PROJECT_DIR" --file "$1"
`,

		"pre-commit": `#!/bin/sh
# pulsarules_codex pre-commit hook - runs governance checks on staged changes.
# ` + marker.Installed + `; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" governance --project "$PROJECT_DIR" --scope commit
`,

		"pre-push": `#!/bin/sh
# pulsarules_codex pre-push hook - runs governance checks before pushing.
# ` + marker.Installed + `; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" governance --project "$PROJECT_DIR"
`,
	}
}

func InstallBinary(dir string) error {
	dst := filepath.Join(dir, ".git", "hooks", "pulsarules_cli")
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

func HookNames() []string {
	names := make([]string, 0, len(hookScripts()))
	for name := range hookScripts() {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func Uninstall(dir string) (removed, restored []string, err error) {
	hooksDir := filepath.Join(dir, ".git", "hooks")
	for name := range hookScripts() {
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
