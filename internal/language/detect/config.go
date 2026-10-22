package detect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

const ConfigFileName = ".pulsarules-languages.json"

type configFile struct {
	Languages []string `json:"languages"`
}

func configPath(repoDir string) string {
	// Meant to be committed, unlike .mcp.json, so the team and CI resolve the same skills.
	return filepath.Join(repoDir, ConfigFileName)
}

func LoadConfig(repoDir string) (set Set, ok bool, err error) {
	var raw []byte
	raw, err = disk.ReadOptional(configPath(repoDir))
	if err != nil {
		return Set{}, false, err
	}
	// No config yet returns ok false, so Resolve falls through to scan.
	if raw == nil {
		return Set{}, false, nil
	}
	var cfg configFile
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return Set{}, false, fmt.Errorf("parse %q: %w", configPath(repoDir), err)
	}
	langs := make([]Language, len(cfg.Languages))
	for idx, name := range cfg.Languages {
		langs[idx] = Language(name)
	}
	return NewSet(langs...), true, nil
}

func SaveConfig(repoDir string, set Set) error {
	if err := jsondoc.Save(configPath(repoDir), configFile{Languages: set.Strings()}); err != nil {
		return fmt.Errorf("save %q: %w", configPath(repoDir), err)
	}
	return nil
}

func RemoveConfig(repoDir string) (removed bool, err error) {
	if err = os.Remove(configPath(repoDir)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("remove %q: %w", configPath(repoDir), err)
	}
	return true, nil
}
