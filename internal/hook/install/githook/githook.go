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

// binaryName is the installer binary co-located with the hook scripts in the
// repository's shared hooks dir, which is where each script looks for it.
const binaryName = "pulsarules_cli"

// hookMode is the tightest mode that still lets git execute the hook script.
const hookMode = 0o500

// Install writes the selected git hook scripts into the repository's shared hooks
// dir. A script already there that Install didn't write (no marker.Installed) is
// renamed to a numbered backup slot rather than destroyed: a git hook isn't
// tracked by git, so its content would otherwise be unrecoverable. backedUp
// reports each rename as a ready-to-print message.
func Install(dir string, hooks []string, opts Options) (backedUp []string, err error) {
	if len(hooks) == 0 {
		return nil, nil
	}
	dest := hooksDir(dir)
	if err = os.MkdirAll(dest, fsperm.DirPrivate); err != nil {
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
		// A git hook must stay executable, so 0o500 is deliberate; it is
		// already the tightest mode that still lets git run the script.
		note, installErr := marker.InstallFile(path, []byte(script), hookMode)
		if installErr != nil {
			return backedUp, fmt.Errorf("install hook %q: %w", name, installErr)
		}
		if note != "" {
			backedUp = append(backedUp, note)
		}
	}
	return backedUp, nil
}

// InstallBinary copies the running installer binary into the repository's
// shared hooks dir so the hook scripts can invoke it. A failure degrades to a
// no-op hook.
func InstallBinary(dir string) error {
	dst := filepath.Join(hooksDir(dir), binaryName)
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

// HookNames returns the list of supported git hook names, sorted for stable
// diagnostics.
func HookNames() []string {
	names := make([]string, 0, len(hookSpecs))
	for name := range hookSpecs {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Uninstall removes every git hook script Install could have written from the
// shared hooks dir, plus the installer binary. Only a hook carrying
// marker.Installed is removed - a git hook is un-mergeable - and removing one
// restores any backup Install left behind; absent/foreign hooks are skipped.
func Uninstall(dir string) (removed, restored []string, err error) {
	dest := hooksDir(dir)
	for name := range hookSpecs {
		path := filepath.Join(dest, name)
		removedOK, note, uninstallErr := marker.UninstallFile(path)
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
