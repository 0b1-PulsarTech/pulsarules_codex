package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestUnwireClaudeMCP_NoNoteWhenNothingChanged(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seedMCP string
	}{
		{name: "absent .mcp.json"},
		{name: "present .mcp.json with no gopls entry", seedMCP: `{"mcpServers": {"other": {}}}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			repoDir := t.TempDir()
			if testCase.seedMCP != "" {
				path := filepath.Join(repoDir, ".mcp.json")
				if err := os.WriteFile(path, []byte(testCase.seedMCP), 0o600); err != nil {
					t.Fatalf("seed .mcp.json: %v", err)
				}
			}

			var report Report
			if err := unwireClaudeMCP(repoDir, &report); err != nil {
				t.Fatalf("unwireClaudeMCP: %v", err)
			}
			if len(report.Notes) != 0 {
				t.Errorf("Notes = %v, want none (nothing was unwired)", report.Notes)
			}
		})
	}
}

func TestUnwireClaudeMCP_NotesWhenGoplsRemoved(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	seed := `{"mcpServers": {"gopls": {"command": "gopls", "args": ["mcp"]}}}`
	path := filepath.Join(repoDir, ".mcp.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed .mcp.json: %v", err)
	}

	var report Report
	if err := unwireClaudeMCP(repoDir, &report); err != nil {
		t.Fatalf("unwireClaudeMCP: %v", err)
	}
	if !strings.Contains(strings.Join(report.Notes, "\n"), "unwired gopls") {
		t.Errorf("Notes = %v, want an 'unwired gopls' entry", report.Notes)
	}
}

//nolint:paralleltest // fakeGoplsOnPath needs t.Setenv, which bars t.Parallel.
func TestGenerateGoplsSkill_ForeignBackup(t *testing.T) {
	fakeGoplsOnPath(t)
	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}

	skillsDir := t.TempDir()
	goplsDir := filepath.Join(skillsDir, "gopls-navigation")
	const mine = "# my own gopls notes, with no ownership marker\n"
	seedFile(t, filepath.Join(goplsDir, "SKILL.md"), mine)

	var report Report
	if err = generateGoplsSkill(templates, skillsDir, &report); err != nil {
		t.Fatalf("generateGoplsSkill: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("no backup warning reported for a foreign gopls-navigation skill")
	}
	assertNote(t, report.Warnings, "gopls-navigation")
	if !dirHoldsContent(t, goplsDir, mine) {
		t.Error("the user's own SKILL.md content is gone from the directory")
	}
}

func dirHoldsContent(tb testing.TB, dir, content string) bool {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		body, readErr := os.ReadFile(filepath.Join(dir, entry.Name())) //nolint:gosec // fixture.
		if readErr == nil && string(body) == content {
			return true
		}
	}
	return false
}

func fakeGoplsOnPath(tb testing.TB) {
	tb.Helper()
	dir := tb.TempDir()
	script := "#!/bin/sh\necho fake gopls instructions\n"
	if err := os.WriteFile(filepath.Join(dir, "gopls"), []byte(script), 0o700); err != nil {
		tb.Fatalf("write fake gopls: %v", err)
	}
	tb.Setenv("PATH", dir)
}
