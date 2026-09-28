package mcpwire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
)

//nolint:paralleltest // fakeOnPath needs t.Setenv, which bars t.Parallel.
func TestWriteMCP_FreshFile(t *testing.T) {
	fakeOnPath(t, "gopls")
	repoDir := t.TempDir()

	wired, skipped, err := WriteMCP(fakeTemplates(), repoDir, langdetect.NewSet(langdetect.Go))
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

	languages := langdetect.NewSet(langdetect.Go)
	if _, _, err := WriteMCP(fakeTemplates(), repoDir, languages); err != nil {
		t.Fatalf("WriteMCP #1: %v", err)
	}
	if _, _, err := WriteMCP(fakeTemplates(), repoDir, languages); err != nil {
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

	if _, _, err := WriteMCP(
		fakeTemplates(),
		repoDir,
		langdetect.NewSet(langdetect.Go),
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
		languages langdetect.Set
		want      []string
	}{
		{
			name:      "dart only wires dart, not gopls",
			languages: langdetect.NewSet(langdetect.Dart),
			want:      []string{"dart"},
		},
		{
			name:      "a language with no managed server wires nothing",
			languages: langdetect.NewSet(langdetect.Rust),
		},
		{
			name:      "go+dart wires both",
			languages: langdetect.NewSet(langdetect.Go, langdetect.Dart),
			want:      []string{"dart", "gopls"},
		},
	}
	//nolint:paralleltest // shares the outer fakeOnPath PATH stub; not parallel-safe.
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repoDir := t.TempDir()
			wired, _, err := WriteMCP(fakeTemplates(), repoDir, testCase.languages)
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

	wired, skipped, err := WriteMCP(fakeTemplates(), repoDir, langdetect.NewSet(langdetect.Go))
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
	Command string `json:"command"`
	Cwd     string `json:"cwd"`
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
