package loader

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type Layered struct {
	rules projectfile.Rules
}

func New(rules projectfile.Rules) *Layered {
	return &Layered{rules: rules}
}

func (layered *Layered) Load(root, explicit, presetFlag string) (*projectfile.File, error) {
	file, err := layered.load(root, explicit, presetFlag)
	return file, capReport(err, file)
}

func (layered *Layered) load(root, explicit, presetFlag string) (*projectfile.File, error) {
	if explicit != "" {
		return loadFile(explicit, explicit, layered.rules)
	}
	if root == "" {
		return nil, nil
	}
	return layered.loadLayers(root, vcs.CommonDir, presetFlag)
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

func (layered *Layered) ForRoot(root string) (*config.GovernanceConfig, []finding.Finding) {
	file, err := layered.Load(root, "", "")
	if err == nil {
		var cfg *config.GovernanceConfig
		if cfg, err = Build(file, ""); err == nil {
			return cfg, nil
		}
	}
	hit := config.Finding(faultFile(err, file), err.Error(), "")
	return withPreset(nil, ""), []finding.Finding{hit}
}
