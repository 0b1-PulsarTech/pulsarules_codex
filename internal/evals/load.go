package evals

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
)

//go:embed data/*.json
var dataFS embed.FS

func Load() ([]Scenario, error) {
	entries, err := fs.Glob(dataFS, "data/*.json")
	if err != nil {
		return nil, fmt.Errorf("glob eval data: %w", err)
	}

	var scenarios []Scenario
	for _, entry := range entries {
		fromFile, loadErr := loadFile(entry)
		if loadErr != nil {
			return nil, loadErr
		}
		scenarios = append(scenarios, fromFile...)
	}

	slices.SortFunc(scenarios, func(left, right Scenario) int {
		if bySkill := cmp.Compare(left.Skill, right.Skill); bySkill != 0 {
			return bySkill
		}
		return cmp.Compare(left.ID, right.ID)
	})
	return scenarios, nil
}

func loadFile(entry string) ([]Scenario, error) {
	raw, err := fs.ReadFile(dataFS, entry)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", entry, err)
	}

	var file scenarioFile
	if decodeErr := json.Unmarshal(raw, &file); decodeErr != nil {
		return nil, fmt.Errorf("decode %s: %w", entry, decodeErr)
	}

	for idx := range file.Scenarios {
		// A scenario that names no skill inherits the one named once at the top.
		if file.Scenarios[idx].Skill == "" {
			file.Scenarios[idx].Skill = file.Skill
		}
	}
	return file.Scenarios, nil
}
