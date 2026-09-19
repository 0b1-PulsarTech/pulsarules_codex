package opencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func UnwireConfig(projectDir string) (changed bool, err error) {
	path := filepath.Join(projectDir, configFile)
	var existing []byte
	existing, err = disk.ReadOptional(path)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return false, fmt.Errorf("%w: %q: %w", jsondoc.ErrUnparseableJSON, path, err)
	}

	var anyChanged bool
	anyChanged, err = stripOwned(config, path)
	if err != nil {
		return false, err
	}
	if !anyChanged {
		return false, nil
	}
	removeOwnedSchema(config)
	if _, err = jsondoc.SaveOrRemove(path, config); err != nil {
		return false, fmt.Errorf("write opencode config: %w", err)
	}
	return true, nil
}

func stripOwned(config map[string]json.RawMessage, path string) (bool, error) {
	instructionsChanged, err := jsondoc.StripSliceSection(
		config, "instructions", fmt.Sprintf("%q instructions", path), stripInstructions,
	)
	if err != nil {
		return false, fmt.Errorf("strip instructions: %w", err)
	}
	var mcpChanged bool
	mcpChanged, err = jsondoc.StripMapSection(
		config,
		"mcp",
		fmt.Sprintf("%q mcp", path),
		stripManagedMCP,
	)
	if err != nil {
		return false, fmt.Errorf("strip mcp: %w", err)
	}
	var skillsChanged bool
	skillsChanged, err = stripSkills(config, path)
	if err != nil {
		return false, fmt.Errorf("strip skills: %w", err)
	}
	return instructionsChanged || mcpChanged || skillsChanged, nil
}

func removeOwnedSchema(config map[string]json.RawMessage) {
	if len(config) != 1 {
		return
	}
	raw, ok := config["$schema"]
	if !ok {
		return
	}
	// Only the exact URL WireConfig sets is ours to remove.
	want, err := json.Marshal(schemaURL)
	if err != nil {
		return
	}
	if bytes.Equal(bytes.TrimSpace(raw), want) {
		delete(config, "$schema")
	}
}

func stripInstructions(instructions *[]string) bool {
	before := len(*instructions)
	globs := instructionGlobs()
	*instructions = slices.DeleteFunc(*instructions, func(entry string) bool {
		return slices.Contains(globs, entry) ||
			entry == legacyAgentsInstructionGlob || entry == legacySkillsInstructionGlob
	})
	return len(*instructions) != before
}

func stripManagedMCP(servers map[string]json.RawMessage) bool {
	if _, present := servers["gopls"]; !present {
		return false
	}
	delete(servers, "gopls")
	return true
}

func stripSkills(config map[string]json.RawMessage, path string) (bool, error) {
	raw, ok := config["skills"]
	if !ok {
		return false, nil
	}
	skills := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &skills); err != nil {
		return false, fmt.Errorf("%w: %q skills: %w", jsondoc.ErrUnparseableJSON, path, err)
	}

	changed, err := jsondoc.StripSliceSection(
		skills, "paths", fmt.Sprintf("%q skills.paths", path), stripSkillsPaths,
	)
	if err != nil {
		return false, fmt.Errorf("strip skills paths: %w", err)
	}
	if !changed {
		return false, nil
	}
	if len(skills) == 0 {
		delete(config, "skills")
		return true, nil
	}
	marshaled, err := json.Marshal(skills)
	if err != nil {
		return false, fmt.Errorf("marshal skills: %w", err)
	}
	config["skills"] = marshaled
	return true, nil
}

func stripSkillsPaths(paths *[]string) bool {
	before := len(*paths)
	*paths = slices.DeleteFunc(*paths, func(entry string) bool { return entry == SkillsSubdir })
	return len(*paths) != before
}
