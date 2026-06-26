package opencodewire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// configFile is the opencode project config merged by WireConfig.
const configFile = "opencode.json"

// schemaURL is opencode's config schema, set when the file is created.
const schemaURL = "https://opencode.ai/config.json"

// instructionGlobs point opencode at the AGENTS.md and the rendered skills so it
// loads them as instructions.
var instructionGlobs = []string{".opencode/AGENTS.md", SkillsSubdir + "/*/SKILL.md"}

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
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's project dir.
	if err != nil && !os.IsNotExist(err) {
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

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode.json: %w", err)
	}
	//nolint:gosec // path is under the caller's project dir.
	if err = os.WriteFile(path, append(out, '\n'), fsperm.FilePrivate); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

// mergeInstructions unions the standards instruction globs into the existing
// instructions array without duplicating.
func mergeInstructions(config map[string]json.RawMessage) error {
	var instructions []string
	if raw, ok := config["instructions"]; ok {
		if err := json.Unmarshal(raw, &instructions); err != nil {
			return fmt.Errorf("parse existing instructions: %w", err)
		}
	}
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
