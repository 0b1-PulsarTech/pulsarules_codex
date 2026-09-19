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
	existing, err := disk.ReadOptional(path)
	if err != nil {
		return err
	}

	config := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err = json.Unmarshal(existing, &config); err != nil {
			return fmt.Errorf("parse existing opencode.json: %w", err)
		}
	}
	if _, ok := config["$schema"]; !ok {
		config["$schema"], _ = json.Marshal(schemaURL)
	}
	if err = mergeInstructions(config); err != nil {
		return fmt.Errorf("merge instructions: %w", err)
	}
	if err = mergeSkills(config); err != nil {
		return fmt.Errorf("merge skills: %w", err)
	}
	if gopls == WithGopls {
		if err = mergeMCP(config); err != nil {
			return fmt.Errorf("merge gopls mcp: %w", err)
		}
	}

	if err = jsondoc.Save(path, config); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	return nil
}

func mergeInstructions(config map[string]json.RawMessage) error {
	var instructions []string
	if raw, ok := config["instructions"]; ok {
		if err := json.Unmarshal(raw, &instructions); err != nil {
			return fmt.Errorf("parse existing instructions: %w", err)
		}
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
	raw, err := json.Marshal(instructions)
	if err != nil {
		return fmt.Errorf("marshal instructions: %w", err)
	}
	config["instructions"] = raw
	return nil
}

func mergeMCP(config map[string]json.RawMessage) error {
	servers := map[string]json.RawMessage{}
	if raw, ok := config["mcp"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return fmt.Errorf("parse existing mcp: %w", err)
		}
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

	raw, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("marshal mcp: %w", err)
	}
	config["mcp"] = raw
	return nil
}
