package opencodewire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsonconfig"
)

// configFile is the opencode project config merged by WireConfig.
const configFile = "opencode.json"

// SkillsSubdir is where rendered skills live in an opencode project, relative
// to the project root. It is also the glob root wired into opencode.json
// instructions.
const SkillsSubdir = ".opencode/skills"

// schemaURL is opencode's config schema, set when the file is created.
const schemaURL = "https://opencode.ai/config.json"

// instructionGlobs point opencode at the root AGENTS.md and the rendered
// skills so it loads them as instructions. AGENTS.md lives at the project
// root (agentswire.WriteAgents), not under .opencode, so every AI-coding
// host that reads only a repo-root AGENTS.md finds the same file opencode
// does.
var instructionGlobs = []string{"AGENTS.md", SkillsSubdir + "/*/SKILL.md"}

// legacyAgentsInstructionGlob is the pre-migration instructions entry for
// the old .opencode/AGENTS.md location, before AGENTS.md moved to the
// project root. mergeInstructions drops it on every install so a reinstall
// converges on "AGENTS.md" instead of carrying both, and unwireInstructions
// matches it too so uninstall clears it even if RetireLegacyAgents never ran.
const legacyAgentsInstructionGlob = ".opencode/AGENTS.md"

// GoplsWiring selects whether WireConfig also merges the gopls MCP server
// entry into opencode.json, replacing a withGopls bool flag argument.
type GoplsWiring string

const (
	// WithGopls tells WireConfig to merge the gopls MCP server entry.
	WithGopls GoplsWiring = "with-gopls"
	// WithoutGopls tells WireConfig to leave the mcp block untouched.
	WithoutGopls GoplsWiring = "without-gopls"
)

// WireConfig merges the standards wiring into <projectDir>/opencode.json: the
// $schema, the instruction globs, and (when gopls is WithGopls) the gopls MCP
// server. It preserves every other key, existing instructions, and other MCP
// servers, and is idempotent.
func WireConfig(projectDir string, gopls GoplsWiring) error {
	path := filepath.Join(projectDir, configFile)
	existing, err := jsonconfig.Read(path)
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

// mergeInstructions unions the standards instruction globs into the existing
// instructions array without duplicating, and retires the legacy
// .opencode/AGENTS.md entry a pre-migration install may have left behind
// (see legacyAgentsInstructionGlob) so a reinstall converges on one AGENTS.md
// entry instead of leaving both.
func mergeInstructions(config map[string]json.RawMessage) error {
	var instructions []string
	if raw, ok := config["instructions"]; ok {
		if err := json.Unmarshal(raw, &instructions); err != nil {
			return fmt.Errorf("parse existing instructions: %w", err)
		}
	}
	instructions = slices.DeleteFunc(instructions, func(entry string) bool {
		return entry == legacyAgentsInstructionGlob
	})
	for _, glob := range instructionGlobs {
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

// mergeGoplsMCP sets the gopls server in the opencode mcp block, preserving any
// other servers.
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
