package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestWriteMCP_FreshFile(t *testing.T) {
	fakeOnPath(t, "gopls")
	repoDir := t.TempDir()

	wired, skipped, err := Write(repoDir, detect.NewSet(detect.Go))
	if err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	if want := []string{"gopls"}; !slices.Equal(wired, want) {
		t.Errorf("wired = %v, want %v", wired, want)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	gopls := readServer(t, repoDir, "gopls")
	if gopls.Command != "gopls" {
		t.Errorf("command = %q, want gopls", gopls.Command)
	}
	if gopls.Cwd != repoDir {
		t.Errorf("cwd = %q, want %q", gopls.Cwd, repoDir)
	}
}

func TestRenderServers_EscapesRepoDir(t *testing.T) {
	t.Parallel()

	servers := []server{{Name: "gopls", Command: "gopls", Args: []string{"mcp"}}}
	testCases := []struct {
		name    string
		repoDir string
	}{
		{name: "plain path", repoDir: "/home/dev/p"},
		{name: "windows backslashes", repoDir: `C:\Users\dev\p`},
		{name: "embedded quote", repoDir: `a"b`},
		{name: "html characters", repoDir: "a<b>&c"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rendered, err := renderServers(testCase.repoDir, servers)
			if err != nil {
				t.Fatalf("renderServers: %v", err)
			}
			var got mcpServer
			if err = json.Unmarshal(rendered["gopls"], &got); err != nil {
				t.Fatalf("rendered entry is not valid JSON: %v", err)
			}
			if got.Cwd != testCase.repoDir {
				t.Errorf("cwd = %q, want %q", got.Cwd, testCase.repoDir)
			}
			if got.Command != "gopls" || !slices.Equal(got.Args, []string{"mcp"}) {
				t.Errorf("entry = %+v, want gopls mcp", got)
			}
		})
	}
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestWriteMCP_PreservesExisting(t *testing.T) {
	fakeOnPath(t, "gopls")
	repoDir := t.TempDir()
	seed := `{
  "someTopLevel": {"keep": true},
  "mcpServers": {
    "other": {"command": "other-mcp", "args": ["serve"], "cwd": "/x"}
  }
}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	languages := detect.NewSet(detect.Go)
	if _, _, err := Write(repoDir, languages); err != nil {
		t.Fatalf("WriteMCP #1: %v", err)
	}
	if _, _, err := Write(repoDir, languages); err != nil {
		t.Fatalf("WriteMCP #2: %v", err)
	}

	raw, err := os.ReadFile(path) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var config struct {
		SomeTopLevel json.RawMessage            `json:"someTopLevel"`
		MCPServers   map[string]json.RawMessage `json:"mcpServers"`
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(config.SomeTopLevel) == 0 {
		t.Error("lost unrelated top-level key someTopLevel")
	}
	for _, name := range []string{"other", "gopls"} {
		if _, ok := config.MCPServers[name]; !ok {
			t.Errorf("missing server %q", name)
		}
	}
	if len(config.MCPServers) != 2 {
		t.Errorf("server count = %d, want 2 (no duplication)", len(config.MCPServers))
	}
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestWriteMCP_GitignoresMCPJSON(t *testing.T) {
	fakeOnPath(t, "gopls")
	repoDir := t.TempDir()

	if _, _, err := Write(
		repoDir,
		detect.NewSet(detect.Go),
	); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	giContent, err := os.ReadFile(filepath.Join(repoDir, ".gitignore")) //nolint:gosec // temp dir.
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(giContent), ".mcp.json") {
		t.Errorf(".gitignore missing .mcp.json entry, got %q", giContent)
	}
}

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestWriteMCP_LanguageAware(t *testing.T) {
	fakeOnPath(t, "gopls", "dart")

	testCases := []struct {
		name      string
		languages detect.Set
		want      []string
	}{
		{
			name:      "dart only wires dart, not gopls",
			languages: detect.NewSet(detect.Dart),
			want:      []string{"dart"},
		},
		{
			name:      "a language with no managed server wires nothing",
			languages: detect.NewSet(detect.Rust),
		},
		{
			name:      "go+dart wires both",
			languages: detect.NewSet(detect.Go, detect.Dart),
			want:      []string{"dart", "gopls"},
		},
	}
	//nolint:paralleltest // shares the outer fakeOnPath PATH stub; not parallel-safe.
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repoDir := t.TempDir()
			wired, _, err := Write(repoDir, testCase.languages)
			if err != nil {
				t.Fatalf("WriteMCP: %v", err)
			}
			got := slices.Clone(wired)
			slices.Sort(got)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("wired = %v, want %v", got, testCase.want)
			}
			if len(testCase.want) == 0 {
				if _, statErr := os.Stat(filepath.Join(repoDir, ".mcp.json")); statErr == nil {
					t.Error("expected no .mcp.json to be written when nothing was wired")
				}
			}
		})
	}
}

func TestWriteMCP_SkipsMissingCommand(t *testing.T) {
	t.Setenv("PATH", "")
	repoDir := t.TempDir()

	wired, skipped, err := Write(repoDir, detect.NewSet(detect.Go))
	if err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	if len(wired) != 0 {
		t.Errorf("wired = %v, want none", wired)
	}
	if want := []string{"gopls"}; !slices.Equal(skipped, want) {
		t.Errorf("skipped = %v, want %v", skipped, want)
	}
	if _, statErr := os.Stat(filepath.Join(repoDir, ".mcp.json")); statErr == nil {
		t.Error("expected no .mcp.json to be written when everything was skipped")
	}
}

type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

func readServer(tb testing.TB, repoDir, name string) mcpServer {
	tb.Helper()
	raw, err := os.ReadFile(filepath.Join(repoDir, ".mcp.json")) //nolint:gosec // temp dir.
	if err != nil {
		tb.Fatalf("read .mcp.json: %v", err)
	}
	var config struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
	}
	if err = json.Unmarshal(raw, &config); err != nil {
		tb.Fatalf("parse .mcp.json: %v", err)
	}
	server, ok := config.MCPServers[name]
	if !ok {
		tb.Fatalf("missing server %q in %+v", name, config.MCPServers)
	}
	return server
}
