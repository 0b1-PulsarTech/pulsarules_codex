package opencode

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpserver"
)

func TestWireConfig_FreshFile(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	if err := WireConfig(projectDir, testServers()); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}
	config := readConfig(t, projectDir)

	if config.Schema != schemaURL {
		t.Errorf("$schema = %q, want %q", config.Schema, schemaURL)
	}
	for _, glob := range instructionGlobs() {
		if !slices.Contains(config.Instructions, glob) {
			t.Errorf("instructions missing %q, got %v", glob, config.Instructions)
		}
	}
	if slices.Contains(config.Instructions, legacySkillsInstructionGlob) {
		t.Errorf(
			"instructions must never carry the eager glob %q, got %v",
			legacySkillsInstructionGlob, config.Instructions,
		)
	}
	if !slices.Contains(config.Skills.Paths, SkillsSubdir) {
		t.Errorf("skills.paths missing %q, got %v", SkillsSubdir, config.Skills.Paths)
	}
	wantCommands := map[string][]string{"gopls": {"gopls", "mcp"}, "dart": {"dart", "mcp-server"}}
	for name, wantCommand := range wantCommands {
		server, ok := config.MCP[name]
		if !ok {
			t.Fatalf("missing %s mcp server, got %v", name, config.MCP)
		}
		if server.Type != "local" || !server.Enabled || !slices.Equal(server.Command, wantCommand) {
			t.Errorf("%s server = %+v, want local enabled %v", name, server, wantCommand)
		}
	}
}

func TestWireConfig_PreservesExisting(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{
  "theme": "dark",
  "instructions": ["CONTRIBUTING.md"],
  "skills": {"paths": ["/abs/path/to/skills"]},
  "mcp": {"other": {"type": "local", "command": ["other"], "enabled": true}}
}`
	path := filepath.Join(projectDir, "opencode.json")
	seedFile(t, path, seed)

	wireTwice(t, projectDir)

	if _, ok := readTopLevel(t, path)["theme"]; !ok {
		t.Error("lost unrelated key theme")
	}

	config := readConfig(t, projectDir)
	want := append([]string{"CONTRIBUTING.md"}, instructionGlobs()...)
	if len(config.Instructions) != len(want) {
		t.Errorf("instructions = %v, want %v", config.Instructions, want)
	}
	wantPaths := []string{"/abs/path/to/skills", SkillsSubdir}
	if len(config.Skills.Paths) != len(wantPaths) {
		t.Errorf("skills.paths = %v, want %v", config.Skills.Paths, wantPaths)
	}
	for _, name := range []string{"other", "gopls"} {
		if _, ok := config.MCP[name]; !ok {
			t.Errorf("missing mcp server %q", name)
		}
	}
}

func TestWireConfig_LegacyInstructionEntry(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"instructions": ["` + legacyAgentsInstructionGlob + `"]}`
	path := filepath.Join(projectDir, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WireConfig(projectDir, nil); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if slices.Contains(config.Instructions, legacyAgentsInstructionGlob) {
		t.Errorf("legacy entry %q survived: %v", legacyAgentsInstructionGlob, config.Instructions)
	}
	if !slices.Contains(config.Instructions, "AGENTS.md") {
		t.Errorf("missing current AGENTS.md entry: %v", config.Instructions)
	}
	count := 0
	for _, entry := range config.Instructions {
		if entry == "AGENTS.md" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("AGENTS.md appears %d times, want exactly 1: %v", count, config.Instructions)
	}
}

func TestWireConfig_LegacySkillsGlob(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	seed := `{"instructions": ["AGENTS.md", "` + legacySkillsInstructionGlob + `"]}`
	path := filepath.Join(projectDir, "opencode.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WireConfig(projectDir, nil); err != nil {
		t.Fatalf("WireConfig: %v", err)
	}

	config := readConfig(t, projectDir)
	if slices.Contains(config.Instructions, legacySkillsInstructionGlob) {
		t.Errorf("eager glob %q survived: %v", legacySkillsInstructionGlob, config.Instructions)
	}
	if len(config.Instructions) != 1 || config.Instructions[0] != "AGENTS.md" {
		t.Errorf("instructions = %v, want [AGENTS.md]", config.Instructions)
	}
	if !slices.Contains(config.Skills.Paths, SkillsSubdir) {
		t.Errorf("skills.paths missing %q, got %v", SkillsSubdir, config.Skills.Paths)
	}
}

func TestWireConfig_NoServersLeavesMCPAlone(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seed    string
		wantMCP string
	}{
		{name: "no mcp key is added", seed: `{"theme":"dark"}`},
		{name: "a corrupt mcp section is not read", seed: `{"mcp":[]}`, wantMCP: `[]`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, configFile)
			if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if err := WireConfig(projectDir, nil); err != nil {
				t.Fatalf("WireConfig: %v", err)
			}
			raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			var fields map[string]json.RawMessage
			if err = json.Unmarshal(raw, &fields); err != nil {
				t.Fatalf("parse: %v", err)
			}
			mcp, present := fields["mcp"]
			if present != (testCase.wantMCP != "") || string(mcp) != testCase.wantMCP {
				t.Errorf("mcp = %q (present %v), want %q", mcp, present, testCase.wantMCP)
			}
		})
	}
}

func TestWireConfig_ParseFailureWording(t *testing.T) {
	t.Parallel()

	const objectCause = "json: cannot unmarshal array into Go value of type " +
		"map[string]jsontext.Value"
	const listCause = "json: cannot unmarshal object into Go value of type []string"
	testCases := []struct {
		name string
		seed string
		want string
	}{
		{
			name: "file not an object",
			seed: `[1]`,
			want: "parse existing opencode.json: " + objectCause,
		},
		{
			name: "instructions of the wrong type", seed: `{"instructions":{}}`,
			want: "merge instructions: parse existing instructions: " + listCause,
		},
		{
			name: "skills of the wrong type", seed: `{"skills":[]}`,
			want: "merge skills: parse existing skills: " + objectCause,
		},
		{
			name: "skills paths of the wrong type", seed: `{"skills":{"paths":{}}}`,
			want: "merge skills: parse existing skills paths: " + listCause,
		},
		{
			name: "mcp of the wrong type", seed: `{"mcp":[]}`,
			want: "merge gopls mcp: parse existing mcp: " + objectCause,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, configFile)
			if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			err := WireConfig(projectDir, testServers())
			if !errors.Is(err, jsondoc.ErrUnparseable) || err.Error() != testCase.want {
				t.Fatalf("err = %v, want ErrUnparseable with text %q", err, testCase.want)
			}
		})
	}
}

func TestWireConfig_NullInputIsWired(t *testing.T) {
	t.Parallel()

	for _, seed := range []string{`null`, `{"skills":null}`, `{"mcp":null}`} {
		t.Run(seed, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, configFile)
			if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if err := WireConfig(projectDir, testServers()); err != nil {
				t.Fatalf("WireConfig(%s): %v", seed, err)
			}
			config := readConfig(t, projectDir)
			if !slices.Contains(config.Skills.Paths, SkillsSubdir) || len(config.MCP) != 2 {
				t.Errorf("config = %+v, want skills.paths and both mcp servers", config)
			}
		})
	}
}

func testServers() []mcpserver.Server {
	return []mcpserver.Server{
		{Name: "gopls", Command: "gopls", Args: []string{"mcp"}},
		{Name: "dart", Command: "dart", Args: []string{"mcp-server"}},
	}
}

type opencodeConfig struct {
	Schema       string               `json:"$schema"`
	Instructions []string             `json:"instructions"`
	MCP          map[string]mcpServer `json:"mcp"`
	Skills       skillsSection        `json:"skills"`
}

type mcpServer struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Enabled bool     `json:"enabled"`
}

type skillsSection struct {
	Paths []string `json:"paths"`
	URLs  []string `json:"urls"`
}

func readConfig(tb testing.TB, projectDir string) opencodeConfig {
	tb.Helper()
	raw, err := os.ReadFile(filepath.Join(projectDir, "opencode.json")) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read opencode.json: %v", err)
	}
	var config opencodeConfig
	if err = json.Unmarshal(raw, &config); err != nil {
		tb.Fatalf("parse opencode.json: %v", err)
	}
	return config
}

func seedFile(tb testing.TB, path, content string) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		tb.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", path, err)
	}
}

func wireTwice(tb testing.TB, projectDir string) {
	tb.Helper()

	for range 2 {
		if err := WireConfig(projectDir, testServers()); err != nil {
			tb.Fatalf("WireConfig: %v", err)
		}
	}
}

func readTopLevel(tb testing.TB, path string) map[string]json.RawMessage {
	tb.Helper()

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read: %v", err)
	}
	var top map[string]json.RawMessage
	if err = json.Unmarshal(raw, &top); err != nil {
		tb.Fatalf("parse: %v", err)
	}
	return top
}
