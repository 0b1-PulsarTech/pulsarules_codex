package loader

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const configAnalyzerID = "config"

func configRules() projectfile.Rules {
	return projectfile.Rules{
		AnalyzerIDs:      registry.ConfigurableIDs(),
		ReservedIDs:      append(registry.IDs(), configAnalyzerID),
		Params:           registry.Params(),
		DelegatedFormats: registry.DelegatedFormats(),
	}
}

func Load(root, explicit, presetFlag string) (*projectfile.File, error) {
	file, err := load(root, explicit, presetFlag)
	return file, capReport(err, file)
}

func load(root, explicit, presetFlag string) (*projectfile.File, error) {
	if explicit != "" {
		return loadFile(explicit, explicit, configRules())
	}
	if root == "" {
		return nil, nil
	}
	return loadLayers(root, vcs.CommonDir, presetFlag)
}

func loadFile(path, display string, rules projectfile.Rules) (*projectfile.File, error) {
	content, err := readBounded(path)
	if err != nil {
		return nil, &fileError{File: display, Err: fmt.Errorf("%s: %w", display, err)}
	}
	// Parse errors already lead with the file and field path.
	file, err := projectfile.Parse(display, content, rules)
	if err != nil {
		return nil, &fileError{File: display, Err: err}
	}
	return file, nil
}

func loadOptional(path, display string, rules projectfile.Rules) (*projectfile.File, error) {
	file, err := loadFile(path, display, rules)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return file, err
}

func Build(file *projectfile.File, presetFlag string) (*config.GovernanceConfig, error) {
	cfg := withPreset(file, presetFlag)
	if file == nil {
		return cfg, nil
	}
	if err := file.ApplyTo(cfg); err != nil {
		return nil, capReport(err, file)
	}
	return cfg, nil
}

func withPreset(file *projectfile.File, presetFlag string) *config.GovernanceConfig {
	cfg := config.Defaults()
	var filePreset string
	if file != nil {
		filePreset = file.Preset
	}
	cfg.Preset = cmp.Or(presetFlag, filePreset, config.PresetRecommended)
	cfg.ApplyPreset()
	return cfg
}

func ForRoot(root string) (*config.GovernanceConfig, []finding.Finding) {
	file, err := Load(root, "", "")
	if err == nil {
		var cfg *config.GovernanceConfig
		if cfg, err = Build(file, ""); err == nil {
			return cfg, nil
		}
	}
	findingItem := finding.Finding{
		AnalyzerID: configAnalyzerID,
		Severity:   finding.SeverityError,
		Category:   finding.CatProject,
		File:       faultFile(err, file),
		Message:    err.Error(),
	}
	return withPreset(nil, ""), []finding.Finding{findingItem}
}
