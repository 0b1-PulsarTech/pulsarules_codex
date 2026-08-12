package opencode

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func TestUnwireConfig_RemovesOurWiring(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}

	if err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	path := filepath.Join(projectDir, configFile)
	if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed once emptied, stat err = %v", path, statErr)
	}
}

func TestUnwireConfig_KeepsSchema(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "$schema": "` + schemaURL + `",
  "instructions": ["CONTRIBUTING.md", "AGENTS.md", ".opencode/skills/*/SKILL.md"]
}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := UnwireConfig(projectDir); err != nil {
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

func TestUnwireConfig_KeepsForeignSchema(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "$schema": "https://example.com/other.json",
  "instructions": ["AGENTS.md", ".opencode/skills/*/SKILL.md"]
}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if config.Schema != "https://example.com/other.json" {
		t.Errorf("foreign $schema should survive, got %q", config.Schema)
	}
}

func TestUnwireConfig_PreservesUnrelated(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "theme": "dark",
  "instructions": ["CONTRIBUTING.md", "AGENTS.md", ".opencode/skills/*/SKILL.md"],
  "mcp": {
    "other": {"type": "local", "command": ["other"], "enabled": true},
    "gopls": {"type": "local", "command": ["gopls", "mcp"], "enabled": true}
  }
}`
	path := filepath.Join(projectDir, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := UnwireConfig(projectDir); err != nil {
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

func TestUnwireConfig_LegacyInstructionEntry(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"instructions": ["CONTRIBUTING.md", "` + legacyAgentsInstructionGlob + `"]}`
	path := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if len(config.Instructions) != 1 || config.Instructions[0] != "CONTRIBUTING.md" {
		t.Errorf("instructions = %v, want [CONTRIBUTING.md]", config.Instructions)
	}
}

func TestUnwireConfig_Unparseable(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	path := filepath.Join(projectDir, "opencode.json")
	original := "{not valid json"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := UnwireConfig(projectDir)
	if !errors.Is(err, jsondoc.ErrUnparseableJSON) {
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

func TestUnwireConfig_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	if err := UnwireConfig(t.TempDir()); err != nil {
		t.Fatalf("UnwireConfig: %v", err)
	}
}

func TestUnwireConfig_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, WithGopls); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}
	if err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig #1: %v", err)
	}
	if err := UnwireConfig(projectDir); err != nil {
		t.Fatalf("UnwireConfig #2: %v", err)
	}
}
