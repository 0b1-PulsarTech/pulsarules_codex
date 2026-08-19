package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestUnwireClaudeMCP_NoNoteWhenNothingChanged is the regression test for the
// bug where unwireClaudeMCP printed "unwired gopls from %s" whenever
// mcpwire.RemoveMCP returned a nil error, even though RemoveMCP returns nil
// for both an absent .mcp.json and a present one carrying no gopls entry -
// neither case actually unwired anything.
func TestUnwireClaudeMCP_NoNoteWhenNothingChanged(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seedMCP string // empty means no .mcp.json is written at all
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

// TestUnwireClaudeMCP_NotesWhenGoplsRemoved asserts the note still fires when
// RemoveMCP actually strips a gopls entry, so the fix does not simply
// silence the note entirely.
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

// TestGenerateGoplsSkill_ReportsBackingUpAForeignSkill covers the one write path in this package
// that used to make a backup and tell nobody. Every other call site threads output.WriteDoc's
// backedUp list into the report; this one discarded it to keep a signature shared with
// opencodeTarget, so a user's own hand-written gopls-navigation/SKILL.md was renamed away while the
// install printed only "generated gopls-navigation skill".
func TestGenerateGoplsSkill_ReportsBackingUpAForeignSkill(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("knowledge.Load: %v", err)
	}

	skillsDir := t.TempDir()
	goplsDir := filepath.Join(skillsDir, "gopls-navigation")
	if err = os.MkdirAll(goplsDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const mine = "# my own gopls notes, with no ownership marker\n"
	skillPath := filepath.Join(goplsDir, "SKILL.md")
	if err = os.WriteFile(skillPath, []byte(mine), 0o600); err != nil {
		t.Fatalf("seed foreign skill: %v", err)
	}

	// why: exercised through mcpwire directly rather than generateGoplsSkill, because the latter
	// shells out to a real `gopls mcp -instructions`; the reporting seam under test is the same.
	backedUp, err := mcpwire.GenerateGoplsSkill(templates, skillsDir, "instructions body")
	if err != nil {
		t.Fatalf("GenerateGoplsSkill: %v", err)
	}
	if len(backedUp) == 0 {
		t.Fatal("no backup reported for a foreign gopls-navigation skill")
	}

	var report Report
	for _, msg := range backedUp {
		report.Warn("%s", msg)
	}
	if len(report.Warnings) != len(backedUp) {
		t.Errorf("warnings = %d, want %d", len(report.Warnings), len(backedUp))
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), "gopls-navigation") {
		t.Errorf("warnings = %v, want them to name the backed-up skill", report.Warnings)
	}

	// The user's content must still exist somewhere on disk, not be gone.
	entries, err := os.ReadDir(goplsDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var found bool
	for _, entry := range entries {
		body, readErr := os.ReadFile(filepath.Join(goplsDir, entry.Name()))
		if readErr == nil && string(body) == mine {
			found = true
		}
	}
	if !found {
		t.Error("the user's own SKILL.md content is gone from the directory")
	}
}
