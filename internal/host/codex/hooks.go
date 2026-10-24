package codex

import (
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func InstallHook(
	templates fs.FS,
	codexDir string,
	loc Location,
) (ownedfile.Ledger, error) {
	return hookset.Install(templates, codexDir, loc.scriptValues(filepath.ListSeparator))
}

func UninstallHook(codexDir string) (ownedfile.Ledger, error) {
	return hookset.Uninstall(codexDir)
}

func WireHooks(codexDir string, loc Location) error {
	if err := refuseSymlink(filepath.Join(codexDir, HooksFile)); err != nil {
		return err
	}
	return hookset.Wire(codexDir, HooksFile, loc.block())
}

func UnwireHooks(codexDir string) (bool, error) {
	if err := refuseSymlink(filepath.Join(codexDir, HooksFile)); err != nil {
		return false, err
	}
	return hookset.Unwire(codexDir, HooksFile)
}

func RenderHooksBlock(loc Location) ([]byte, error) {
	return hookset.RenderBlock(loc.block())
}
