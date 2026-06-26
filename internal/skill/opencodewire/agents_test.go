package opencodewire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestWriteAgents asserts the AGENTS.md is written under .opencode with the
// project name, the opencode skills dir, and the skill list filled in.
func TestWriteAgents(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	projectDir := t.TempDir()
	if err = WriteAgents(fakeTemplates(), projectDir, idx); err != nil {
		t.Fatalf("WriteAgents: %v", err)
	}

	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(projectDir, ".opencode", "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(raw)
	for _, want := range []string{
		filepath.Base(projectDir),
		SkillsSubdir,
		"`project-router`",
		"`git-history`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AGENTS.md missing %q:\n%s", want, got)
		}
	}
}
