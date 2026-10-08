package marker

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func literalGatedAssets() []string {
	return []string{
		"hooks/skill-router-reminder.sh.tmpl",
		"hooks/README.md",
		"docs/AGENTS.md.tmpl",
		"hooks/opencode-plugin.js",
		"skills/gopls-navigation.header.md",
	}
}

func skillsTemplateFamily(t *testing.T, templates fs.FS) []string {
	t.Helper()

	mdFiles, err := fs.Glob(templates, "skills/*.md.tmpl")
	if err != nil {
		t.Fatalf("glob skills/*.md.tmpl: %v", err)
	}
	mdcFiles, err := fs.Glob(templates, "skills/*.mdc.tmpl")
	if err != nil {
		t.Fatalf("glob skills/*.mdc.tmpl: %v", err)
	}
	return append(mdFiles, mdcFiles...)
}

func loadGatedFixture(t *testing.T) (*knowledge.Index, fs.FS, *render.Renderer) {
	t.Helper()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	return idx, templates, rnd
}

func assertCarriesMarker(t *testing.T, label, body string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("render %s: %v", label, err)
	}
	if !strings.Contains(body, Installed) {
		t.Errorf("rendered %s does not carry marker.Installed", label)
	}
}

func TestGatedAssetsCarryInstalled(t *testing.T) {
	t.Parallel()

	idx, templates, rnd := loadGatedFixture(t)
	if family := skillsTemplateFamily(t, templates); len(family) != 5 {
		t.Fatalf(
			"skills/ template family is %v (len %d), want 5 (SKILL.md.tmpl, WORKFLOW.md.tmpl, "+
				"router.md.tmpl, SKILL.mdc.tmpl, router.mdc.tmpl); a member was "+
				"added or removed - "+
				"update this test's expectations before trusting it again",
			family, len(family),
		)
	}
	if len(idx.Skills) == 0 {
		t.Fatal("no skills loaded to render")
	}
	if len(idx.Workflows) == 0 {
		t.Fatal("no workflows loaded to render")
	}

	for _, skill := range idx.Skills {
		t.Run("skill/"+skill.ID, func(t *testing.T) {
			t.Parallel()
			body, renderErr := rnd.RenderSkill(idx, skill, nil)
			assertCarriesMarker(t, "skill "+skill.ID, body, renderErr)
		})
		t.Run("skill-cursor/"+skill.ID, func(t *testing.T) {
			t.Parallel()
			body, renderErr := rnd.RenderSkillCursor(idx, skill, nil)
			assertCarriesMarker(t, "cursor rule "+skill.ID, body, renderErr)
		})
	}
	for _, workflow := range idx.Workflows {
		t.Run("workflow/"+workflow.ID, func(t *testing.T) {
			t.Parallel()
			body, renderErr := rnd.RenderWorkflow(idx, workflow)
			assertCarriesMarker(t, "workflow "+workflow.ID, body, renderErr)
		})
	}
}

func TestLiteralGatedAssetsCarryInstalled(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	for _, path := range literalGatedAssets() {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			content, readErr := fs.ReadFile(templates, path)
			if readErr != nil {
				t.Fatalf("read %q: %v", path, readErr)
			}
			if !strings.Contains(string(content), Installed) {
				t.Errorf("%q does not carry marker.Installed:\n%s", path, content)
			}
		})
	}
}

func pathSet(paths []string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, path := range paths {
		set[path] = true
	}
	return set
}

func flagUnlisted(t *testing.T, templates fs.FS, family, listed map[string]bool) fs.WalkDirFunc {
	t.Helper()

	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || family[path] {
			return nil
		}
		content, readErr := fs.ReadFile(templates, path)
		if readErr != nil {
			return fmt.Errorf("read %q: %w", path, readErr)
		}
		if strings.Contains(string(content), Installed) && !listed[path] {
			t.Errorf("%q carries marker.Installed but is missing from literalGatedAssets", path)
		}
		return nil
	}
}

func TestLiteralGatedAssetsListIsComplete(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	listed := pathSet(literalGatedAssets())
	family := pathSet(skillsTemplateFamily(t, templates))
	family["skills/parts.tmpl"] = true

	walkErr := fs.WalkDir(templates, ".", flagUnlisted(t, templates, family, listed))
	if walkErr != nil {
		t.Fatalf("walk embedded templates: %v", walkErr)
	}
}
