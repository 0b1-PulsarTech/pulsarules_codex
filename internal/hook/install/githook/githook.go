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

const binaryName = "pulsarules_cli"

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
		const hookMode = 0o500
		// The tightest mode that still lets git execute the hook script.
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

func Uninstall(dir string) (removed, restored, orphaned []string, err error) {
	dest := hooksDir(dir)
	paths := make([]string, 0, len(hookSpecs()))
	for name := range hookSpecs() {
		path := filepath.Join(dest, name)
		paths = append(paths, path)
		removedOK, note, uninstallErr := marker.UninstallFile(path)
		if uninstallErr != nil {
			return removed, restored, orphaned, fmt.Errorf(
				"uninstall hook %q: %w", name, uninstallErr,
			)
		}
		if !removedOK {
			continue
		}
		removed = append(removed, name)
		if note != "" {
			restored = append(restored, note)
		}
	}
	// Queried before the binary goes: a numbered slot survives Restore and
	// would otherwise sit in the shared hooks dir unmentioned forever.
	if orphaned, err = marker.OrphanNotes(paths...); err != nil {
		return removed, restored, orphaned, err
	}
	if err = os.RemoveAll(filepath.Join(dest, binaryName)); err != nil {
		return removed, restored, orphaned, fmt.Errorf("remove installer binary: %w", err)
	}
	return removed, restored, orphaned, nil
}
