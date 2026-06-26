package mcpwire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateGoplsSkill asserts the generated SKILL.md combines the curated
// header with the live instructions under the reference heading.
func TestGenerateGoplsSkill(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	const instructions = "# The gopls MCP server\n\nUse go_search to find symbols.\n"
	if err := GenerateGoplsSkill(fakeTemplates(), skillsDir, instructions); err != nil {
		t.Fatalf("GenerateGoplsSkill: %v", err)
	}

	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(skillsDir, "gopls-navigation", "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated skill: %v", err)
	}
	got := string(raw)
	for _, want := range []string{
		"name: gopls-navigation",
		"## Full gopls MCP reference",
		"Use go_search to find symbols.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated skill missing %q:\n%s", want, got)
		}
	}
}

// TestGenerateGoplsSkill_Gitignore asserts the generated skill dir gets the
// same sibling .gitignore every other installed skill gets.
func TestGenerateGoplsSkill_Gitignore(t *testing.T) {
	t.Parallel()

	skillsDir := t.TempDir()
	if err := GenerateGoplsSkill(fakeTemplates(), skillsDir, "instructions"); err != nil {
		t.Fatalf("GenerateGoplsSkill: %v", err)
	}
	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(skillsDir, "gopls-navigation", ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if want := "SKILL.md\n.gitignore\n"; string(raw) != want {
		t.Errorf(".gitignore content = %q, want %q", raw, want)
	}
}

// TestGoplsOnPath_Absent asserts gopls is reported missing when PATH is empty.
func TestGoplsOnPath_Absent(t *testing.T) {
	t.Setenv("PATH", "")
	if GoplsOnPath() {
		t.Error("GoplsOnPath() = true with empty PATH, want false")
	}
}
