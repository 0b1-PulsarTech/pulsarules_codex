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
		body, renderErr := rnd.RenderSkill(idx, skill)
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
		if strings.Contains(body, "Apply every applicable rule") {
			t.Errorf("%q: rendered skill still carries the generic checklist footer", skill.ID)
		}
	}

	router, err := rnd.RenderRouter(idx)
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
