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

// binaryRel is the path to the installer binary relative to the worktree root,
// co-located in .git/hooks/ alongside the hook scripts.
const binaryRel = ".git/hooks/pulsarules_cli"

// hookMode is the tightest mode that still lets git execute the hook script.
const hookMode = 0o500

// Install writes the selected git hook scripts into dir/.git/hooks/. A
// script already there that Install didn't write (no marker.Installed) is
// renamed to a numbered ".pulsarules-backup" slot rather than destroyed,
// since a git hook isn't tracked by git - its content would otherwise be
// unrecoverable. backedUp reports each rename as a ready-to-print message.
func Install(dir string, hooks []string) (backedUp []string, err error) {
	if len(hooks) == 0 {
		return nil, nil
	}
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err = os.MkdirAll(hooksDir, fsperm.DirPrivate); err != nil {
		return nil, fmt.Errorf("mkdir .git/hooks: %w", err)
	}
	for _, name := range hooks {
		script, ok := hookScripts[name]
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
				return backedUp, fmt.Errorf("%w", backupErr)
			}
			backedUp = append(backedUp, marker.BackupMessage(path, backupPath))
		} else {
			// Remove an owned (or absent) file first; os.WriteFile with
			// O_TRUNC fails on read-only files left by a previous install
			// (0o500). A foreign file above was already moved out of the
			// way by Backup, so this branch never touches one.
			// why: path is the trusted project hooks dir this installer maintains.
			_ = os.Remove(path)
		}
		// A git hook must stay executable, so 0o500 is deliberate; it is
		// already the tightest mode that still lets git run the script.
		if writeErr := os.WriteFile(path, []byte(script), hookMode); writeErr != nil {
			return backedUp, fmt.Errorf("write .git/hooks/%s: %w", name, writeErr)
		}
	}
	return backedUp, nil
}

// hookScripts maps git hook name to shell script content. Each script forwards
// to the installer binary located inside .pulsarules/bin/.
var hookScripts = map[string]string{
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

// InstallBinary copies the running installer binary into dir/.git/hooks/ so the
// hook scripts can invoke it. A failure degrades to a no-op hook.
func InstallBinary(dir string) error {
	dst := filepath.Join(dir, ".git", "hooks", "pulsarules_cli")
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

// HookNames returns the list of supported git hook names, sorted for stable
// diagnostics.
func HookNames() []string {
	names := make([]string, 0, len(hookScripts))
	for name := range hookScripts {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Uninstall removes every git hook script Install could have written from
// dir/.git/hooks/, plus the installer binary. Only a hook carrying
// marker.Installed is removed - a git hook is un-mergeable, so this never
// deletes a hand-authored one. Removing an owned hook restores any backup
// Install left behind (see marker.Backup); absent/foreign hooks are skipped.
func Uninstall(dir string) (removed, restored []string, err error) {
	hooksDir := filepath.Join(dir, ".git", "hooks")
	for name := range hookScripts {
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
			return removed, restored, fmt.Errorf("%w", restoreErr)
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
