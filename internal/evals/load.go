package evals

import (
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
)

// dataFS holds the committed eval scenario files, one per skill under
// evaluation. Embedded (not under knowledge/standards) since these files
// drive validation and grading, not a rendered SKILL.md.
//
//go:embed data/*.json
var dataFS embed.FS

// Load reads every committed data/<skill>.json file and returns their
// scenarios flattened into one slice, sorted by skill id then scenario id for
// deterministic output.
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

	slices.SortFunc(scenarios, func(a, b Scenario) int {
		if bySkill := cmp.Compare(a.Skill, b.Skill); bySkill != 0 {
			return bySkill
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return scenarios, nil
}

// loadFile decodes one data/<skill>.json file and defaults each scenario's
// Skill to the file-level skill when the scenario left it unset.
func loadFile(entry string) ([]Scenario, error) {
	raw, err := fs.ReadFile(dataFS, entry)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", entry, err)
	}

	var file scenarioFile
	if decodeErr := json.Unmarshal(raw, &file); decodeErr != nil {
		return nil, fmt.Errorf("decode %s: %w", entry, decodeErr)
	}

	for i := range file.Scenarios {
		if file.Scenarios[i].Skill == "" {
			file.Scenarios[i].Skill = file.Skill
		}
	}
	return file.Scenarios, nil
}
