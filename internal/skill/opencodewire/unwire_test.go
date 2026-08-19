package opencodewire

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

// TestUnwireConfig_RemovesOurWiring asserts a fresh WireConfig(WithGopls) is
// fully reversed: once instructions and the gopls server are gone, only
// $schema remains, and since it's exactly WireConfig's value, UnwireConfig
// deletes it too and removes the now-empty opencode.json entirely - a
// tool-created file leaves nothing behind.
func TestUnwireConfig_RemovesOurWiring(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}

	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	path := filepath.Join(projectDir, configFile)
	if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed once emptied, stat err = %v", path, statErr)
	}
}

// TestUnwireConfig_KeepsSchemaWhenOtherContentSurvives asserts $schema is
// left in place when the file still carries a user's own content after
// unwiring, since removeOwnedSchema only fires once nothing else is left.
func TestUnwireConfig_KeepsSchemaWhenOtherContentSurvives(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"$schema": "` + schemaURL + `", "instructions": ["CONTRIBUTING.md", "AGENTS.md", ".opencode/skills/*/SKILL.md"]}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if config.Schema != schemaURL {
		t.Errorf("$schema should survive alongside remaining content, got %q", config.Schema)
	}
	if len(config.Instructions) != 1 || config.Instructions[0] != "CONTRIBUTING.md" {
		t.Errorf("instructions = %v, want [CONTRIBUTING.md]", config.Instructions)
	}
}

// TestUnwireConfig_KeepsForeignSchema asserts a $schema pointed somewhere
// else survives even once the file is otherwise emptied, since that value
// is not one WireConfig could have written.
func TestUnwireConfig_KeepsForeignSchema(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"$schema": "https://example.com/other.json", "instructions": ["AGENTS.md", ".opencode/skills/*/SKILL.md"]}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if config.Schema != "https://example.com/other.json" {
		t.Errorf("foreign $schema should survive, got %q", config.Schema)
	}
}

// TestUnwireConfig_PreservesUnrelated asserts an unrelated top-level key, a
// hand-authored instruction, and an unrelated mcp server all survive.
func TestUnwireConfig_PreservesUnrelated(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "theme": "dark",
  "instructions": ["CONTRIBUTING.md", "AGENTS.md", ".opencode/skills/*/SKILL.md"],
  "mcp": {"other": {"type": "local", "command": ["other"], "enabled": true}, "gopls": {"type": "local", "command": ["gopls", "mcp"], "enabled": true}}
}`
	path := filepath.Join(projectDir, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var top map[string]json.RawMessage
	if err = json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := top["theme"]; !ok {
		t.Error("lost unrelated key theme")
	}

	config := readConfig(t, projectDir)
	if len(config.Instructions) != 1 || config.Instructions[0] != "CONTRIBUTING.md" {
		t.Errorf("instructions = %v, want [CONTRIBUTING.md]", config.Instructions)
	}
	if _, ok := config.MCP["other"]; !ok {
		t.Error("lost unrelated mcp server other")
	}
	if _, ok := config.MCP["gopls"]; ok {
		t.Error("gopls server should have been removed")
	}
}

// TestUnwireConfig_RemovesLegacyInstructionEntry asserts UnwireConfig also
// clears the pre-migration ".opencode/AGENTS.md" instructions entry even
// when the file-level migration (RetireLegacyAgents) never ran - e.g. a
// project that installed before the migration and is being uninstalled
// directly, skipping a reinstall.
func TestUnwireConfig_RemovesLegacyInstructionEntry(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"instructions": ["CONTRIBUTING.md", "` + legacyAgentsInstructionGlob + `"]}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if len(config.Instructions) != 1 || config.Instructions[0] != "CONTRIBUTING.md" {
		t.Errorf("instructions = %v, want [CONTRIBUTING.md]", config.Instructions)
	}
}

// TestUnwireConfig_Unparseable asserts an opencode.json that is not valid
// JSON is left untouched and the error wraps fsx.ErrUnparseableJSON.
func TestUnwireConfig_Unparseable(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	path := filepath.Join(projectDir, "opencode.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := UnwireConfig(projectDir)
	if !errors.Is(err, fsx.ErrUnparseableJSON) {
		t.Fatalf("err = %v, want fsx.ErrUnparseableJSON", err)
	}
	got, readErr := os.ReadFile(path) //nolint:gosec // temp dir.
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != original {
		t.Errorf("file was rewritten: got %q, want unchanged %q", got, original)
	}
}

// TestUnwireConfig_NoOpWhenAbsent asserts a missing opencode.json is not an
// error, and reports changed = false since nothing was there to unwire.
func TestUnwireConfig_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	changed, err := UnwireConfig(t.TempDir())
	if err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}
	if changed {
		t.Error("changed = true, want false (nothing was there to unwire)")
	}
}

// TestUnwireConfig_ChangedSignal is the regression test for the bug where a
// caller could not tell a real unwire from a no-op: UnwireConfig returns a
// nil error for both an absent file and a present file carrying none of
// this tool's wiring, so callers must key off changed, not off a nil error.
func TestUnwireConfig_ChangedSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string // empty means no opencode.json is written at all
		wantChanged bool
	}{
		{name: "absent file", wantChanged: false},
		{name: "present with no wiring", seed: `{"theme": "dark"}`, wantChanged: false},
		{
			name:        "present with instructions wired",
			seed:        `{"instructions": ["AGENTS.md"]}`,
			wantChanged: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			if testCase.seed != "" {
				path := filepath.Join(projectDir, configFile)
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			changed, err := UnwireConfig(projectDir)
			if err != nil {
				t.Fatalf("UnwireConfig: %v", err)
			}
			if changed != testCase.wantChanged {
				t.Errorf("changed = %v, want %v", changed, testCase.wantChanged)
			}
		})
	}
}

// TestUnwireConfig_Idempotent asserts running UnwireConfig twice is not an
// error.
func TestUnwireConfig_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}
	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig #1: %v", err)
	}
	if _, err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig #2: %v", err)
	}
}
