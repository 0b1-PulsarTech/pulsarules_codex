package githook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

// binaryRel is the path to the installer binary relative to the worktree root,
// co-located in .git/hooks/ alongside the hook scripts.
const binaryRel = ".git/hooks/pulsarules_cli"

// hookMode is the tightest mode that still lets git execute the hook script.
const hookMode = 0o500

// Install writes the selected git hook scripts into dir/.git/hooks/. Each hook
// is a small shell script that forwards to the installer binary. Already-
// existing hooks are overwritten.
func Install(dir string, hooks []string) error {
	if len(hooks) == 0 {
		return nil
	}
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir .git/hooks: %w", err)
	}
	for _, name := range hooks {
		script, ok := hookScripts[name]
		if !ok {
			return fmt.Errorf("unknown git hook %q", name)
		}
		path := filepath.Join(hooksDir, name)
		// Remove existing file first; os.WriteFile with O_TRUNC fails on
		// read-only files left by a previous install (0o500).
		//nolint:gosec // trusted project hooks dir.
		_ = os.Remove(path)
		// A git hook must stay executable, so 0o500 is deliberate; it is
		// already the tightest mode that still lets git run the script.
		//nolint:gosec // G306: executable bit is required for a git hook.
		if err := os.WriteFile(path, []byte(script), hookMode); err != nil {
			return fmt.Errorf("write .git/hooks/%s: %w", name, err)
		}
	}
	return nil
}

// hookScripts maps git hook name to shell script content. Each script forwards
// to the installer binary located inside .pulsarules/bin/.
var hookScripts = map[string]string{
	"commit-msg": `#!/bin/sh
# pulsarules_codex commit-msg hook - validates commit message format.
# Installed by pulsarules_cli; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" commitlint --project "$PROJECT_DIR" --file "$1"
`,

	"pre-commit": `#!/bin/sh
# pulsarules_codex pre-commit hook - runs governance checks on staged changes.
# Installed by pulsarules_cli; remove or edit this file to disable.
PROJECT_DIR=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
BINARY="$PROJECT_DIR/` + binaryRel + `"
[ -x "$BINARY" ] || exit 0
exec "$BINARY" governance --project "$PROJECT_DIR" --scope commit
`,

	"pre-push": `#!/bin/sh
# pulsarules_codex pre-push hook - runs governance checks before pushing.
# Installed by pulsarules_cli; remove or edit this file to disable.
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

// HookNames returns the list of supported git hook names.
func HookNames() []string {
	names := make([]string, 0, len(hookScripts))
	for name := range hookScripts {
		names = append(names, name)
	}
	return names
}
