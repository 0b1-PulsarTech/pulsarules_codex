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

func UnwireConfig(projectDir string) (changed bool, err error) {
	path := filepath.Join(projectDir, configFile)
	var existing []byte
	existing, err = jsonconfig.Read(path)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return false, fmt.Errorf("%w: %q: %w", fsx.ErrUnparseableJSON, path, err)
	}

	var instructionsChanged bool
	instructionsChanged, err = fsx.StripSliceSection(
		config, "instructions", fmt.Sprintf("%q instructions", path), stripInstructions,
	)
	if err != nil {
		return false, fmt.Errorf("strip instructions: %w", err)
	}
	var mcpChanged bool
	mcpChanged, err = fsx.StripMapSection(
		config,
		"mcp",
		fmt.Sprintf("%q mcp", path),
		stripGoplsMCP,
	)
	if err != nil {
		return false, fmt.Errorf("strip mcp: %w", err)
	}
	if !instructionsChanged && !mcpChanged {
		return false, nil
	}
	removeOwnedSchema(config)
	if _, err = fsx.SaveOrRemove(path, config); err != nil {
		return false, fmt.Errorf("write opencode config: %w", err)
	}
	return true, nil
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

func stripGoplsMCP(servers map[string]json.RawMessage) bool {
	if _, present := servers["gopls"]; !present {
		return false
	}
	delete(servers, "gopls")
	return true
}
