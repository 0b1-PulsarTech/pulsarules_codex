package render

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestRenderSkill_AllSkills renders every embedded skill and asserts each
// SKILL.md leads with its frontmatter name and the router carries its contract.
func TestRenderSkill_AllSkills(t *testing.T) {
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
			continue // router renders from its own curated template.
		}
		if !strings.Contains(body, "## Mandatory workflow") {
			t.Errorf("%q: rendered skill missing curated ## Mandatory workflow", skill.ID)
		}
		if strings.Contains(body, "Apply every applicable rule") {
			t.Errorf("%q: rendered skill still carries the generic checklist footer", skill.ID)
		}
	}

	router, err := rnd.RenderRouter(idx, nil)
	if err != nil {
		t.Fatalf("RenderRouter: %v", err)
	}
	for _, want := range []string{"MANDATORY orchestrator", "Dispatch table", "Tests are not optional routing"} {
		if !strings.Contains(router, want) {
			t.Errorf("router missing %q", want)
		}
	}
}

// TestRender_Workflow asserts RenderWorkflow produces a WORKFLOW.md that carries
// the workflow name, its steps, and its body.
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

// TestRender_UnknownComposition asserts Render surfaces a composed reference that
// does not resolve in the index, rather than rendering a broken skill.
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
