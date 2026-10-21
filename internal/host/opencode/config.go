package opencode

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

const configFile = "opencode.json"

const schemaURL = "https://opencode.ai/config.json"

func instructionGlobs() []string {
	// Repo root: every host reading AGENTS.md finds opencode's file.
	return []string{"AGENTS.md"}
}

const legacyAgentsInstructionGlob = ".opencode/AGENTS.md"

const legacySkillsInstructionGlob = SkillsSubdir + "/*/SKILL.md"

type GoplsWiring string

const (
	WithGopls    GoplsWiring = "with-gopls"
	WithoutGopls GoplsWiring = "without-gopls"
)

func WireConfig(projectDir string, gopls GoplsWiring) error {
	path := filepath.Join(projectDir, configFile)
	doc, err := jsondoc.Load(path)
	if err != nil {
		return err
	}
	if !doc.Has("$schema") {
		if err = jsondoc.SetSection(doc, "$schema", schemaURL); err != nil {
			return fmt.Errorf("set schema: %w", err)
		}
	}
	if err = mergeInstructions(doc); err != nil {
		return fmt.Errorf("merge instructions: %w", err)
	}
	if err = mergeSkills(doc); err != nil {
		return fmt.Errorf("merge skills: %w", err)
	}
	if gopls == WithGopls {
		if err = mergeMCP(doc); err != nil {
			return fmt.Errorf("merge gopls mcp: %w", err)
		}
	}

	if err = doc.Save(); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	return nil
}

func mergeInstructions(doc *jsondoc.Document) error {
	instructions, err := jsondoc.Section[[]string](doc, "instructions")
	if err != nil {
		return err
	}
	// Retire the legacy globs a prior install left so a reinstall converges.
	instructions = slices.DeleteFunc(instructions, func(entry string) bool {
		return entry == legacyAgentsInstructionGlob || entry == legacySkillsInstructionGlob
	})
	for _, glob := range instructionGlobs() {
		if !slices.Contains(instructions, glob) {
			instructions = append(instructions, glob)
		}
	}
	return jsondoc.SetSection(doc, "instructions", instructions)
}

func mergeMCP(doc *jsondoc.Document) error {
	servers, err := jsondoc.Section[map[string]json.RawMessage](doc, "mcp")
	if err != nil {
		return err
	}
	if servers == nil {
		servers = map[string]json.RawMessage{}
	}
	gopls, err := json.Marshal(map[string]any{
		"type":    "local",
		"command": []string{"gopls", "mcp"},
		"enabled": true,
	})
	if err != nil {
		return fmt.Errorf("marshal gopls mcp: %w", err)
	}
	servers["gopls"] = gopls
	return jsondoc.SetSection(doc, "mcp", servers)
}
