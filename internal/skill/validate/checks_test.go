package validate

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestRouterPresent asserts the mandatory project-router skill is required: the
// embedded index satisfies it, an empty index does not. (SkillExists is backed
// by the loader's by-id map, so the present case uses the real embedded index.)
func TestRouterPresent(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := routerPresent(idx); len(problems) != 0 {
		t.Errorf("embedded index should have the router, got %v", problems)
	}
	if problems := routerPresent(&knowledge.Index{}); len(problems) == 0 {
		t.Error("empty index should report the missing router")
	}
}

// TestSkillCompositions asserts an unresolved composed reference is reported with
// its owner and kind, while a fully-resolving composition is silent.
func TestSkillCompositions(t *testing.T) {
	t.Parallel()

	problems := skillCompositions(&knowledge.Index{Skills: []knowledge.Skill{
		{ID: "x", ComposeRules: []string{"nope"}, ComposeSkills: []string{"ghost"}},
	}})
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %v", problems)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{`skill "x" composes unknown rule "nope"`, `skill "x" composes unknown skill "ghost"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("problems missing %q; got %v", want, problems)
		}
	}
}

// TestReferencesResolve asserts a rule citing an unknown reference is reported and
// the embedded index (whose citations all resolve) is silent.
func TestReferencesResolve(t *testing.T) {
	t.Parallel()

	problems := referencesResolve(&knowledge.Index{Rules: []knowledge.Rule{
		{ID: "r", References: []string{"ghost"}},
	}})
	want := `rule "r" cites unknown reference "ghost"`
	if len(problems) != 1 || !strings.Contains(problems[0], want) {
		t.Fatalf("expected one unknown-reference problem, got %v", problems)
	}

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := referencesResolve(idx); len(problems) != 0 {
		t.Errorf("embedded citations should all resolve, got %v", problems)
	}
}
