package output

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// why: covers two mandatory skills (one of them project-router, whose
// declared compose_skills must not be walked as a dependency), a no-deps
// skill, an a->b->c chain, and an a->b->a cycle.
func dependencyFixtureIndex() *knowledge.Index {
	return &knowledge.Index{Skills: []knowledge.Skill{
		{ID: "project-router", Order: 0, AlwaysLoad: true, ComposeSkills: []string{"solo"}},
		{ID: "go-style", Order: 10, AlwaysLoad: true},
		{ID: "solo", Order: 20},
		{ID: "chain-a", Order: 30, ComposeSkills: []string{"chain-b"}},
		{ID: "chain-b", Order: 31, ComposeSkills: []string{"chain-c"}},
		{ID: "chain-c", Order: 32},
		{ID: "cycle-a", Order: 40, ComposeSkills: []string{"cycle-b"}},
		{ID: "cycle-b", Order: 41, ComposeSkills: []string{"cycle-a"}},
	}}
}

// TestSelectionResolve covers router-only, all, the empty (mandatory-only)
// default, a no-deps skill, a transitive a->b->c chain, and an a->b->a cycle
// that must terminate rather than recurse forever.
func TestSelectionResolve(t *testing.T) {
	t.Parallel()

	idx := dependencyFixtureIndex()
	testCases := []struct {
		name       string
		sel        Selection
		wantIDs    []string
		wantPulled []DependencyPull
	}{
		{
			name:    "router-only yields just the router",
			sel:     Selection{RouterOnly: true},
			wantIDs: []string{"project-router"},
		},
		{
			name:    "empty selection resolves to the mandatory baseline only",
			sel:     Selection{},
			wantIDs: []string{"project-router", "go-style"},
		},
		{
			name: "all yields every skill",
			sel:  Selection{All: true},
			wantIDs: []string{
				"project-router",
				"go-style",
				"solo",
				"chain-a",
				"chain-b",
				"chain-c",
				"cycle-a",
				"cycle-b",
			},
		},
		{
			name:    "a skill with no deps",
			sel:     Selection{IDs: []string{"solo"}},
			wantIDs: []string{"project-router", "go-style", "solo"},
		},
		{
			name:    "a chain a->b->c pulls in every link",
			sel:     Selection{IDs: []string{"chain-a"}},
			wantIDs: []string{"project-router", "go-style", "chain-a", "chain-b", "chain-c"},
			wantPulled: []DependencyPull{
				{Skill: "chain-b", RequiredBy: "chain-a"},
				{Skill: "chain-c", RequiredBy: "chain-b"},
			},
		},
		{
			name:    "a cycle a->b->a terminates",
			sel:     Selection{IDs: []string{"cycle-a"}},
			wantIDs: []string{"project-router", "go-style", "cycle-a", "cycle-b"},
			wantPulled: []DependencyPull{
				{Skill: "cycle-b", RequiredBy: "cycle-a"},
			},
		},
		{
			name:    "project-router's own compose_skills is not a dependency",
			sel:     Selection{IDs: []string{"chain-c"}},
			wantIDs: []string{"project-router", "go-style", "chain-c"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotIDs, gotPulled := testCase.sel.Resolve(idx)
			if !slices.Equal(gotIDs, testCase.wantIDs) {
				t.Errorf("ids = %v, want %v", gotIDs, testCase.wantIDs)
			}
			if !slices.Equal(gotPulled, testCase.wantPulled) {
				t.Errorf("pulled = %v, want %v", gotPulled, testCase.wantPulled)
			}
		})
	}
}
