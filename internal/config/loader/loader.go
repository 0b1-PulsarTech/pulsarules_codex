package loader

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const ConfigAnalyzerID = "config"

func Rules() config.Rules {
	internal := registry.InternalIDs()
	return config.Rules{
		AnalyzerIDs: slices.DeleteFunc(registry.IDs(), func(id string) bool {
			return slices.Contains(internal, id)
		}),
		ReservedIDs:      append(registry.IDs(), ConfigAnalyzerID),
		Params:           registry.Params(),
		DelegatedFormats: registry.DelegatedFormats(),
	}
}

func Load(root, explicit string) (*config.File, error) {
	if explicit != "" {
		return loadFile(explicit, explicit)
	}
	if root == "" {
		return nil, nil
	}
	return loadOptional(filepath.Join(root, config.Name), config.Name)
}

func loadFile(path, display string) (*config.File, error) {
	content, err := readBounded(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", display, err)
	}
	// Parse errors already lead with the file and field path.
	return config.Parse(display, content, Rules())
}

func loadOptional(path, display string) (*config.File, error) {
	file, err := loadFile(path, display)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return file, err
}

func Build(file *config.File, presetFlag string) (*config.GovernanceConfig, error) {
	cfg := withPreset(file, presetFlag)
	if file == nil {
		return cfg, nil
	}
	if err := file.ApplyTo(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func withPreset(file *config.File, presetFlag string) *config.GovernanceConfig {
	cfg := config.Defaults()
	var filePreset string
	if file != nil {
		filePreset = file.Preset
	}
	cfg.Preset = cmp.Or(presetFlag, filePreset, config.PresetRecommended)
	cfg.ApplyPreset()
	return cfg
}

func ForRepo(repo vcs.Repository) (*config.GovernanceConfig, []finding.Finding) {
	if repo == nil {
		return ForRoot("")
	}
	return ForRoot(repo.Root())
}

func ForRoot(root string) (*config.GovernanceConfig, []finding.Finding) {
	file, err := Load(root, "")
	if err == nil {
		var cfg *config.GovernanceConfig
		if cfg, err = Build(file, ""); err == nil {
			return cfg, nil
		}
	}
	findingItem := finding.Finding{
		AnalyzerID: ConfigAnalyzerID,
		Severity:   finding.SeverityError,
		Category:   finding.CatProject,
		File:       config.Name,
		Message:    err.Error(),
	}
	return withPreset(nil, ""), []finding.Finding{findingItem}
}
