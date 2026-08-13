package host

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
