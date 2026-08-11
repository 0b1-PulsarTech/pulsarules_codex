package githook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

const binaryRel = ".git/hooks/pulsarules_cli"

func Install(dir string, hooks []string) error {
	if len(hooks) == 0 {
		return nil
	}
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir .git/hooks: %w", err)
	}
	for _, name := range hooks {
		script, ok := hookScripts()[name]
		if !ok {
			return fmt.Errorf("unknown git hook %q", name)
		}
		path := filepath.Join(hooksDir, name)
		exists, ours, checkErr := ownedfile.Check(path)
		if checkErr != nil {
			return fmt.Errorf("check %q: %w", path, checkErr)
		}
		if exists && !ours {
			if _, backupErr := ownedfile.Backup(path); backupErr != nil {
				return fmt.Errorf("back up %q: %w", path, backupErr)
			}
		} else {
			// An owned or absent file is removed so mode always applies, even over a
			// read-only prior install; the error adds nothing the write would not say.
			_ = os.Remove(path)
		}
		const hookMode = 0o500
		if writeErr := os.WriteFile(path, []byte(script), hookMode); writeErr != nil {
			return fmt.Errorf("write .git/hooks/%s: %w", name, writeErr)
		}
	}
	return nil
}

func hookScripts() map[string]string {
	return map[string]string{
		"commit-msg": `#!/bin/sh
# pulsarules_codex commit-msg hook - validates commit message format.
# ` + ownedfile.Marker + `; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" commitlint --project "$PROJECT_DIR" --file "$1"
`,

		"pre-commit": `#!/bin/sh
# pulsarules_codex pre-commit hook - runs governance checks on staged changes.
# ` + ownedfile.Marker + `; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" governance --project "$PROJECT_DIR" --scope commit
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
