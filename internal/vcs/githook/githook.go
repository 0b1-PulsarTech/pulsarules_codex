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

func Install(dir string, hooks []string, opts Options) (ledger ownedfile.Ledger, err error) {
	if len(hooks) == 0 {
		return ledger, nil
	}
	dest := hooksDir(dir)
	if err = os.MkdirAll(dest, disk.DirPrivate); err != nil {
		return ledger, fmt.Errorf("mkdir %q: %w", dest, err)
	}
	for _, name := range hooks {
		script, ok := hookScript(name, opts)
		if !ok {
			return ledger, fmt.Errorf(
				"unknown git hook %q (want %s)", name, strings.Join(HookNames(), "|"),
			)
		}
		path := filepath.Join(dest, name)
		const hookMode = 0o500
		// The tightest mode that still lets git execute the hook script.
		note, installErr := ownedfile.InstallFile(path, []byte(script), hookMode)
		if installErr != nil {
			return ledger, fmt.Errorf("install hook %q: %w", name, installErr)
		}
		ledger.AddBackup(note)
	}
	return ledger, nil
}

func InstallBinary(dir string) error {
	return selfbin.Install(hooksDir(dir))
}

func HookNames() []string {
	return slices.Sorted(maps.Keys(hookSpecs()))
}

type UninstallResult struct {
	Removed []string
	ownedfile.Ledger
}

func Uninstall(dir string) (UninstallResult, error) {
	var res UninstallResult
	dest := hooksDir(dir)
	specs := hookSpecs()
	paths := make([]string, 0, len(specs))
	for name := range specs {
		paths = append(paths, filepath.Join(dest, name))
	}
	var err error
	// The ledger is read before the binary goes: a numbered slot survives Restore and
	// would otherwise sit in the shared hooks dir unmentioned forever.
	var removed []string
	removed, res.Ledger, err = ownedfile.UninstallFiles(paths...)
	for _, path := range removed {
		res.Removed = append(res.Removed, filepath.Base(path))
	}
	if err != nil {
		return res, err
	}
	if err = selfbin.Remove(dest); err != nil {
		return res, err
	}
	return res, nil
}
