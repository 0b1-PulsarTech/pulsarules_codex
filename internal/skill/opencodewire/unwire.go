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

// UnwireConfig removes the instruction globs and gopls MCP entry WireConfig
// merged into <projectDir>/opencode.json, dropping "instructions"/"mcp"
// once empty. It drops "$schema" only when it is the sole remaining key and
// exactly WireConfig's URL, so a user's own $schema survives; invalid JSON
// or an absent file leaves things untouched (wraps fsx.ErrUnparseableJSON).
// changed reports whether anything on disk actually moved, so a caller can
// tell a real removal from a no-op instead of assuming one from a nil error
// - UnwireConfig returns nil error for both an absent file and a present
// file carrying neither wired instructions nor a gopls entry.
func UnwireConfig(projectDir string) (changed bool, err error) {
	path := filepath.Join(projectDir, configFile)
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's project dir.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	config := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &config); err != nil {
		return false, fmt.Errorf("%w: %q: %w", fsx.ErrUnparseableJSON, path, err)
	}

	instructionsChanged, err := fsx.StripSliceSection(
		config, "instructions", fmt.Sprintf("%q instructions", path), stripInstructions,
	)
	if err != nil {
		return false, fmt.Errorf("strip instructions: %w", err)
	}
	mcpChanged, err := fsx.StripMapSection(
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

// removeOwnedSchema deletes config's "$schema" key when it is the only key
// left in config and its value is exactly schemaURL - the only value
// WireConfig ever sets - so this only fires for a file this tool could have
// both created and now fully unwound, never one still carrying other content
// or a user's own $schema pointed somewhere else.
func removeOwnedSchema(config map[string]json.RawMessage) {
	if len(config) != 1 {
		return
	}
	raw, ok := config["$schema"]
	if !ok {
		return
	}
	want, err := json.Marshal(schemaURL)
	if err != nil {
		return
	}
	if bytes.Equal(bytes.TrimSpace(raw), want) {
		delete(config, "$schema")
	}
}

// stripInstructions drops the standards instruction globs - both the current
// ones and the legacy .opencode/AGENTS.md entry a pre-migration install may
// have left (legacyAgentsInstructionGlob, config.go) - from *instructions in
// place. It reports whether it changed anything.
func stripInstructions(instructions *[]string) bool {
	before := len(*instructions)
	*instructions = slices.DeleteFunc(*instructions, func(entry string) bool {
		return slices.Contains(instructionGlobs, entry) || entry == legacyAgentsInstructionGlob
	})
	return len(*instructions) != before
}

// stripGoplsMCP drops the gopls server from servers, in place. It reports
// whether it changed anything.
func stripGoplsMCP(servers map[string]json.RawMessage) bool {
	if _, present := servers["gopls"]; !present {
		return false
	}
	delete(servers, "gopls")
	return true
}
