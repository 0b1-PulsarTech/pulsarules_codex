package opencode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func UnwireConfig(projectDir string) error {
	path := filepath.Join(projectDir, configFile)
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's project dir.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %q: %w", path, err)
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return fmt.Errorf("%w: %q: %w", jsondoc.ErrUnparseableJSON, path, err)
	}

	instructionsChanged, err := jsondoc.StripSliceSection(
		config, "instructions", fmt.Sprintf("%q instructions", path), stripInstructions,
	)
	if err != nil {
		return fmt.Errorf("strip instructions: %w", err)
	}
	mcpChanged, err := jsondoc.StripMapSection(
		config,
		"mcp",
		fmt.Sprintf("%q mcp", path),
		stripManagedMCP,
	)
	if err != nil {
		return fmt.Errorf("strip mcp: %w", err)
	}
	if !instructionsChanged && !mcpChanged {
		return nil
	}
	removeOwnedSchema(config)
	if _, err = jsondoc.SaveOrRemove(path, config); err != nil {
		return fmt.Errorf("write opencode config: %w", err)
	}
	return nil
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
	*instructions = slices.DeleteFunc(*instructions, func(entry string) bool {
		return slices.Contains(instructionGlobs(), entry) || entry == legacyAgentsInstructionGlob
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
