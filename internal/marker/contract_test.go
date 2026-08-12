package marker

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// literalGatedAssets are the embedded templates copied - or text/template
// rendered outside the skills/ family's shared render.Renderer namespace -
// verbatim into a file some marker.Check call site inspects for ownership.
// Deriving this set automatically would need a call-graph trace, for every
// marker.Check call site across hookwire/opencodehook/agentswire/mcpwire, of
// which template path feeds the file each one checks; that is not attempted
// here. TestLiteralGatedAssetsListIsComplete is the fallback this leans on:
// it fails the day a template gains marker.Installed without being added
// here, so the list cannot go stale unnoticed the way the old hardcoded
// table did.
var literalGatedAssets = []string{
	"hooks/skill-router-reminder.sh",    // hookwire.InstallHook (internal/skill/hookwire/hooks.go)
	"hooks/README.md",                   // hookwire.InstallHook (internal/skill/hookwire/hooks.go)
	"docs/AGENTS.md.tmpl",               // agentswire.Write (internal/skill/agentswire/agents.go)
	"hooks/opencode-plugin.js",          // opencodehook.Install (internal/hook/install/opencodehook/opencodehook.go)
	"skills/gopls-navigation.header.md", // mcpwire.GenerateGoplsSkill (internal/skill/mcpwire/gopls.go)
}

// skillsTemplateFamily returns the skills/ templates render.Renderer executes
// as a whole top-level document (namespace.go's topTemplates): every
// *.md.tmpl and *.mdc.tmpl directly under skills/. skills/parts.tmpl is
// excluded on purpose - it holds only {{define}} partials (including the
// "mdcFrontmatter" block the two .mdc members compose from) and produces no
// content when executed under its own name. TestGatedAssetsCarryInstalled
// renders every member of this family; its length assertion there catches a
// new member immediately instead of the coverage silently going stale.
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

// TestGatedAssetsCarryInstalled is the drift test binding Installed to the
// real embedded skills/ template family (SKILL.md.tmpl, WORKFLOW.md.tmpl,
// router.md.tmpl and their .mdc cursor counterparts, see
// skillsTemplateFamily). Every currently loaded skill and workflow, plus
// project-router's own richer template, is actually RENDERED through
// render.Renderer - not grepped as raw template source - so a marker line
// hiding behind a composed partial (the .mdc frontmatter) is proven present
// in what actually lands on disk, and rewording the marker anywhere in the
// family fails this test regardless of which skill or workflow renders it.
func TestGatedAssetsCarryInstalled(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}

	if family := skillsTemplateFamily(t, templates); len(family) != 5 {
		t.Fatalf(
			"skills/ template family is %v (len %d), want 5 (SKILL.md.tmpl, WORKFLOW.md.tmpl, "+
				"router.md.tmpl, SKILL.mdc.tmpl, router.mdc.tmpl); a member was added or removed - "+
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
			if renderErr != nil {
				t.Fatalf("render %q: %v", skill.ID, renderErr)
			}
			if !strings.Contains(body, Installed) {
				t.Errorf("rendered skill %q does not carry marker.Installed", skill.ID)
			}
		})
		t.Run("skill-cursor/"+skill.ID, func(t *testing.T) {
			t.Parallel()
			body, renderErr := rnd.RenderSkillCursor(idx, skill, nil)
			if renderErr != nil {
				t.Fatalf("render cursor rule %q: %v", skill.ID, renderErr)
			}
			if !strings.Contains(body, Installed) {
				t.Errorf("rendered cursor rule %q does not carry marker.Installed", skill.ID)
			}
		})
	}
	for _, workflow := range idx.Workflows {
		t.Run("workflow/"+workflow.ID, func(t *testing.T) {
			t.Parallel()
			body, renderErr := rnd.RenderWorkflow(idx, workflow)
			if renderErr != nil {
				t.Fatalf("render workflow %q: %v", workflow.ID, renderErr)
			}
			if !strings.Contains(body, Installed) {
				t.Errorf("rendered workflow %q does not carry marker.Installed", workflow.ID)
			}
		})
	}
}

// TestLiteralGatedAssetsCarryInstalled asserts every hand-listed
// literalGatedAssets entry carries Installed verbatim in its raw embedded
// content, or a reworded header would silently stop the remover that checks
// it from recognising its own file while uninstall still reports success.
func TestLiteralGatedAssetsCarryInstalled(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	for _, path := range literalGatedAssets {
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

// TestLiteralGatedAssetsListIsComplete fails when some embedded template
// outside the skills/ render family carries marker.Installed without being
// listed in literalGatedAssets, so that hand-listed set - which
// skillsTemplateFamily's automatic derivation cannot cover, see its doc
// comment - cannot silently drift out of sync with reality the way the
// original hardcoded table did.
func TestLiteralGatedAssetsListIsComplete(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	listed := make(map[string]bool, len(literalGatedAssets))
	for _, path := range literalGatedAssets {
		listed[path] = true
	}
	family := make(map[string]bool, 6)
	for _, path := range skillsTemplateFamily(t, templates) {
		family[path] = true
	}
	family["skills/parts.tmpl"] = true // shared partial the family composes from, not a standalone asset

	walkErr := fs.WalkDir(templates, ".", func(path string, entry fs.DirEntry, err error) error {
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
	})
	if walkErr != nil {
		t.Fatalf("walk embedded templates: %v", walkErr)
	}
}
