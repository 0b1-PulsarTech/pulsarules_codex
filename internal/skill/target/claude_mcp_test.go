package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
		report.warn("%s", msg)
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
