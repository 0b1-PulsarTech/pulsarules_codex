package opencode

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func UnwireConfig(projectDir string) (changed bool, err error) {
	path := filepath.Join(projectDir, configFile)
	var doc *jsondoc.Document
	doc, err = jsondoc.Load(path)
	if err != nil {
		return false, err
	}
	if !doc.Exists() {
		return false, nil
	}

	var anyChanged bool
	anyChanged, err = stripOwned(doc)
	if err != nil {
		return false, err
	}
	if !anyChanged {
		return false, nil
	}
	removeOwnedSchema(doc)
	if _, err = doc.SaveOrRemove(); err != nil {
		return false, fmt.Errorf("write opencode config: %w", err)
	}
	return true, nil
}

func stripOwned(doc *jsondoc.Document) (bool, error) {
	instructionsChanged, err := jsondoc.StripSlice(doc, "instructions", stripInstructions)
	if err != nil {
		return false, fmt.Errorf("strip instructions: %w", err)
	}
	var mcpChanged bool
	mcpChanged, err = jsondoc.StripMap(doc, "mcp", stripManagedMCP)
	if err != nil {
		return false, fmt.Errorf("strip mcp: %w", err)
	}
	var skillsChanged bool
	skillsChanged, err = stripSkills(doc)
	if err != nil {
		return false, fmt.Errorf("strip skills: %w", err)
	}
	return instructionsChanged || mcpChanged || skillsChanged, nil
}

func removeOwnedSchema(doc *jsondoc.Document) {
	if doc.Len() != 1 {
		return
	}
	// Only the exact URL WireConfig sets is ours to remove.
	schema, err := jsondoc.Section[string](doc, "$schema")
	if err == nil && schema == schemaURL {
		doc.Delete("$schema")
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

func stripSkills(doc *jsondoc.Document) (bool, error) {
	if !doc.Has("skills") {
		return false, nil
	}
	skills, err := doc.Child("skills")
	if err != nil {
		return false, err
	}
	changed, err := jsondoc.StripSlice(skills, "paths", stripSkillsPaths)
	if err != nil {
		return false, fmt.Errorf("strip skills paths: %w", err)
	}
	if !changed {
		return false, nil
	}
	if err = doc.PutChild("skills", skills); err != nil {
		return false, err
	}
	return true, nil
}

func stripSkillsPaths(paths *[]string) bool {
	before := len(*paths)
	*paths = slices.DeleteFunc(*paths, func(entry string) bool { return entry == SkillsSubdir })
	return len(*paths) != before
}
