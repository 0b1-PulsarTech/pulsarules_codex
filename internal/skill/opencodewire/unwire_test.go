package opencodewire

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

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

	if _, err := UnwireConfig(projectDir); err != nil {
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

type skillsCase struct {
	name       string
	seed       string
	wantPaths  []string
	wantNoKey  bool
	wantURLLen int
}

func assertSkillsConfig(tb testing.TB, config opencodeConfig, testCase skillsCase) {
	tb.Helper()

	if slices.Contains(config.Skills.Paths, SkillsSubdir) {
		tb.Errorf("SkillsSubdir survived: %v", config.Skills.Paths)
	}
	for _, want := range testCase.wantPaths {
		if !slices.Contains(config.Skills.Paths, want) {
			tb.Errorf("skills.paths missing user entry %q, got %v", want, config.Skills.Paths)
		}
	}
	if len(config.Skills.URLs) != testCase.wantURLLen {
		tb.Errorf("skills.urls = %v, want len %d", config.Skills.URLs, testCase.wantURLLen)
	}
}

func TestUnwireConfig_Skills(t *testing.T) {
	t.Parallel()

	testCases := []skillsCase{
		{
			name:      "only our path removes the whole key",
			seed:      `{"skills": {"paths": ["` + SkillsSubdir + `"]}}`,
			wantNoKey: true,
		},
		{
			name:      "user path survives",
			seed:      `{"skills": {"paths": ["/abs/path/to/skills", "` + SkillsSubdir + `"]}}`,
			wantPaths: []string{"/abs/path/to/skills"},
		},
		{
			name: "sibling urls key survives once paths empties",
			seed: `{"skills": {"paths": ["` + SkillsSubdir +
				`"], "urls": ["https://example.com/.well-known/skills/"]}}`,
			wantURLLen: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			path := filepath.Join(projectDir, configFile)
			seedFile(t, path, testCase.seed)

			changed, err := UnwireConfig(projectDir)
			if err != nil {
				t.Fatalf("UnwireConfig: %v", err)
			}
			if !changed {
				t.Fatalf("changed = false, want true")
			}

			if testCase.wantNoKey {
				if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("expected %q to be removed, stat err = %v", path, statErr)
				}
				return
			}
			assertSkillsConfig(t, readConfig(t, projectDir), testCase)
		})
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

	if _, err := UnwireConfig(projectDir); err != nil {
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

func TestUnwireConfig_ChangedSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
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
