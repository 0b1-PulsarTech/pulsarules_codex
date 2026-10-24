package loader

import (
	"cmp"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func privateRules(rules projectfile.Rules, base *projectfile.File) projectfile.Rules {
	if base == nil {
		return rules
	}
	// Clipped once, so every append copies and never writes into the caller's spare capacity.
	ids := slices.Clip(rules.AnalyzerIDs)
	// The private file may tune a command only the versioned file defines.
	for _, entry := range base.Delegated {
		ids = append(ids, entry.ID)
	}
	rules.AnalyzerIDs = ids
	return rules
}

func (layered *Layered) stack(
	user *projectfile.File, files []*projectfile.File, presetFlag string,
) (*projectfile.File, error) {
	// The user layer sits under the project files, which may tune a command only it defines.
	projectRules := privateRules(layered.rules, user)
	var merged *projectfile.File
	for _, file := range files {
		if merged == nil {
			merged = file
			continue
		}
		pair := layers{base: merged, private: file, preset: presetFlag, rules: projectRules}
		var err error
		if merged, err = pair.combine(); err != nil {
			return nil, err
		}
	}
	if user == nil || merged == nil {
		return cmp.Or(merged, user), nil
	}
	pair := layers{base: user, private: merged, preset: presetFlag, rules: layered.rules}
	return pair.underlay()
}

func (layered *Layered) loadUser(root string, rules projectfile.Rules) (*projectfile.File, error) {
	if layered.userConfigDir == "" {
		return nil, nil
	}
	path := filepath.Join(layered.userConfigDir, userDirName, privateFileName)
	return loadOptional(path, displayName(root, path), rules)
}
