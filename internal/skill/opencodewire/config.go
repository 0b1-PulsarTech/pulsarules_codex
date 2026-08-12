package opencodewire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

const configFile = "opencode.json"

const SkillsSubdir = ".opencode/skills"

const schemaURL = "https://opencode.ai/config.json"

func instructionGlobs() []string {
	return []string{"AGENTS.md", SkillsSubdir + "/*/SKILL.md"}
}

const legacyAgentsInstructionGlob = ".opencode/AGENTS.md"

type GoplsWiring string

const (
	WithGopls    GoplsWiring = "with-gopls"
	WithoutGopls GoplsWiring = "without-gopls"
)

func WireConfig(projectDir string, gopls GoplsWiring) error {
	path := filepath.Join(projectDir, configFile)
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's project dir.
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %q: %w", path, err)
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
	if gopls == WithGopls {
		if err = mergeGoplsMCP(config); err != nil {
			return fmt.Errorf("merge gopls mcp: %w", err)
		}
	}

	if err = fsx.Save(path, config); err != nil {
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

func mergeGoplsMCP(config map[string]json.RawMessage) error {
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
