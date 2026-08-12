package render

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRenderSkill_NormativeSection(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	for _, skill := range idx.Skills {
		body, renderErr := rnd.RenderSkill(idx, skill, nil)
		if renderErr != nil {
			t.Errorf("render %q: %v", skill.ID, renderErr)
			continue
		}
		if !strings.HasPrefix(body, "---\nname: "+skill.ID+"\n") {
			t.Errorf("%q: missing frontmatter name", skill.ID)
		}
		if skill.ID == "project-router" {
			continue
		}
		has, normErr := HasNormativeSection(idx, skill)
		if normErr != nil {
			t.Errorf("%q: HasNormativeSection: %v", skill.ID, normErr)
		} else if !has {
			t.Errorf(
				"%q: rendered skill carries no composed must/forbidden/validation section",
				skill.ID,
			)
		}
		if strings.Contains(body, "Apply every applicable rule") {
			t.Errorf("%q: rendered skill still carries the generic checklist footer", skill.ID)
		}
	}

	router, err := rnd.RenderRouter(idx, nil)
	if err != nil {
		t.Fatalf("RenderRouter: %v", err)
	}
	for _, want := range []string{
		"MANDATORY orchestrator", "Dispatch table", "Tests are not optional routing",
	} {
		if !strings.Contains(router, want) {
			t.Errorf("router missing %q", want)
		}
	}
}

func TestComposedLinters_NoDuplicates(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, skill := range idx.Skills {
		t.Run(skill.ID, func(t *testing.T) {
			t.Parallel()
			seen := map[string]bool{}
			for _, linter := range composedLinters(idx, skill) {
				if seen[linter] {
					t.Errorf(
						"linter %q appears more than once in %q's composed linters",
						linter,
						skill.ID,
					)
				}
				seen[linter] = true
			}
		})
	}
}

func TestRender_Workflow(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	wf, ok := idx.Workflow("refactoring")
	if !ok {
		t.Fatal("refactoring workflow not found in index")
	}
	body, renderErr := rnd.RenderWorkflow(idx, wf)
	if renderErr != nil {
		t.Fatalf("RenderWorkflow: %v", renderErr)
	}
	if !strings.HasPrefix(body, "---\nname: refactoring\n") {
		t.Errorf("RenderWorkflow: missing frontmatter, got prefix: %q", body[:min(len(body), 80)])
	}
	for _, want := range []string{"# Refactoring", "## Steps", "put on one hat"} {
		if !strings.Contains(body, want) {
			t.Errorf("RenderWorkflow output missing %q", want)
		}
	}
}

func TestRender_UnknownComposition(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	idx := &knowledge.Index{}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	if _, err := rnd.Render(
		idx,
		knowledge.Skill{ID: "x", ComposeRules: []string{"nope"}},
	); err == nil {
		t.Fatal("expected error for unknown composed rule, got nil")
	}
}

func TestRenderSkillCursor_AllSkills(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	for _, skill := range idx.Skills {
		body, renderErr := rnd.RenderSkillCursor(idx, skill, nil)
		if renderErr != nil {
			t.Errorf("render cursor %q: %v", skill.ID, renderErr)
			continue
		}
		if !strings.HasPrefix(body, "---\ndescription: ") {
			t.Errorf("%q: missing Cursor frontmatter description", skill.ID)
		}
		if !strings.Contains(body, "alwaysApply: false") {
			t.Errorf("%q: expected alwaysApply: false, got:\n%s", skill.ID, body)
		}
		if strings.HasPrefix(body, "---\nname: ") {
			t.Errorf("%q: cursor rule still carries SKILL.md's name field", skill.ID)
		}
	}

	router, err := rnd.RenderCursorRouter(idx, nil)
	if err != nil {
		t.Fatalf("RenderCursorRouter: %v", err)
	}
	for _, want := range []string{
		"MANDATORY orchestrator", "Dispatch table", "alwaysApply: false",
	} {
		if !strings.Contains(router, want) {
			t.Errorf("cursor router missing %q", want)
		}
	}
}
