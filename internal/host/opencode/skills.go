package opencode

import (
	"encoding/json"
	"fmt"
	"slices"
)

const SkillsSubdir = ".opencode/skills"

func mergeSkills(config map[string]json.RawMessage) error {
	skills := map[string]json.RawMessage{}
	if raw, ok := config["skills"]; ok {
		if err := json.Unmarshal(raw, &skills); err != nil {
			return fmt.Errorf("parse existing skills: %w", err)
		}
	}

	var paths []string
	// Wire skills.paths, the lazy index opencode expands only on a tool call; never
	// instructions, which opencode reads in full every session.
	if raw, ok := skills["paths"]; ok {
		if err := json.Unmarshal(raw, &paths); err != nil {
			return fmt.Errorf("parse existing skills paths: %w", err)
		}
	}
	if !slices.Contains(paths, SkillsSubdir) {
		paths = append(paths, SkillsSubdir)
	}

	pathsRaw, err := json.Marshal(paths)
	if err != nil {
		return fmt.Errorf("marshal skills paths: %w", err)
	}
	skills["paths"] = pathsRaw

	skillsRaw, err := json.Marshal(skills)
	if err != nil {
		return fmt.Errorf("marshal skills: %w", err)
	}
	config["skills"] = skillsRaw
	return nil
}
