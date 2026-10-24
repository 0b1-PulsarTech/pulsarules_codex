package codex

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
)

const LegacySkillsSubdir = "skills"

func SettingsWiresHook(codexDir string) (bool, error) {
	body, err := disk.ReadOptional(filepath.Join(codexDir, "settings.json"))
	if err != nil {
		return false, err
	}
	return strings.Contains(string(body), hookset.ScriptName), nil
}
