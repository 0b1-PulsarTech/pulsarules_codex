package claude

import (
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func InstallHook(
	templates fs.FS,
	claudeDir string,
	loc Location,
) (ownedfile.Ledger, error) {
	return hookset.Install(templates, claudeDir, loc.scriptValues(filepath.ListSeparator))
}

func UninstallHook(claudeDir string) (ownedfile.Ledger, error) {
	return hookset.Uninstall(claudeDir)
}

func WireSettings(claudeDir, settingsFile string, loc Location) error {
	return hookset.Wire(claudeDir, settingsFile, loc.block())
}

func UnwireSettings(claudeDir, settingsFile string) (bool, error) {
	return hookset.Unwire(claudeDir, settingsFile)
}

func RenderHooksBlock(loc Location) ([]byte, error) {
	return hookset.RenderBlock(loc.block())
}
