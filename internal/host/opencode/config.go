package opencode

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
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

func WireConfig(projectDir string, servers []mcp.Server) error {
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
	if err = mergeMCP(doc, servers); err != nil {
		return fmt.Errorf("merge mcp: %w", err)
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

type serverEntry struct {
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
	Type    string   `json:"type"`
}

func mergeMCP(doc *jsondoc.Document, servers []mcp.Server) error {
	if len(servers) == 0 {
		return nil
	}
	merged, err := jsondoc.Section[map[string]json.RawMessage](doc, "mcp")
	if err != nil {
		return err
	}
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	for _, srv := range servers {
		entry := serverEntry{
			Type:    "local",
			Command: append([]string{srv.Command}, srv.Args...),
			Enabled: true,
		}
		raw, marshalErr := json.Marshal(entry)
		if marshalErr != nil {
			return fmt.Errorf("marshal %s mcp: %w", srv.Name, marshalErr)
		}
		merged[srv.Name] = raw
	}
	return jsondoc.SetSection(doc, "mcp", merged)
}
