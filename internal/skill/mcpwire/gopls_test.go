package mcpwire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateGoplsSkill_LiveInstructions(t *testing.T) {
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

func TestGoplsOnPath_Absent(t *testing.T) {
	t.Setenv("PATH", "")
	if GoplsOnPath() {
		t.Error("GoplsOnPath() = true with empty PATH, want false")
	}
}
