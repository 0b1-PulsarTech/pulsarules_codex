package selection

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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

func TestChoiceResolve(t *testing.T) {
	t.Parallel()

	idx := dependencyFixtureIndex()
	testCases := []struct {
		name       string
		sel        Choice
		wantIDs    []string
		wantPulled []DependencyPull
	}{
		{
			name:    "router-only yields just the router",
			sel:     Choice{RouterOnly: true},
			wantIDs: []string{"project-router"},
		},
		{
			name:    "empty selection resolves to the mandatory baseline only",
			sel:     Choice{},
			wantIDs: []string{"project-router", "go-style"},
		},
		{
			name: "all yields every skill",
			sel:  Choice{All: true},
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
			sel:     Choice{IDs: []string{"solo"}},
			wantIDs: []string{"project-router", "go-style", "solo"},
		},
		{
			name:    "a chain a->b->c pulls in every link",
			sel:     Choice{IDs: []string{"chain-a"}},
			wantIDs: []string{"project-router", "go-style", "chain-a", "chain-b", "chain-c"},
			wantPulled: []DependencyPull{
				{Skill: "chain-b", RequiredBy: "chain-a"},
				{Skill: "chain-c", RequiredBy: "chain-b"},
			},
		},
		{
			name:    "a cycle a->b->a terminates",
			sel:     Choice{IDs: []string{"cycle-a"}},
			wantIDs: []string{"project-router", "go-style", "cycle-a", "cycle-b"},
			wantPulled: []DependencyPull{
				{Skill: "cycle-b", RequiredBy: "cycle-a"},
			},
		},
		{
			name:    "project-router's own compose_skills is not a dependency",
			sel:     Choice{IDs: []string{"chain-c"}},
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

func languageFixtureIndex(t *testing.T) *knowledge.Index {
	t.Helper()

	idx, err := knowledge.NewIndex([]knowledge.Skill{
		{ID: "project-router", Order: 0, AlwaysLoad: true},
		{ID: "go-style", Order: 10, AlwaysLoad: true, Languages: []string{"go"}},
		{ID: "dart-style", Order: 11, AlwaysLoad: true, Languages: []string{"dart"}},
		{ID: "ts-style", Order: 12, AlwaysLoad: true, Languages: []string{"typescript"}},
		{ID: "web-style", Order: 13, AlwaysLoad: true, Languages: []string{"go", "typescript"}},
		{ID: "code-minimalism", Order: 40, AlwaysLoad: true},
	})
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestChoiceResolve_LanguageAware(t *testing.T) {
	t.Parallel()

	idx := languageFixtureIndex(t)
	every := []string{
		"project-router", "go-style", "dart-style", "ts-style", "web-style", "code-minimalism",
	}
	testCases := []struct {
		name      string
		languages detect.Set
		wantIDs   []string
	}{
		{
			name:      "go-only project drops the dart and typescript skills",
			languages: detect.NewSet(language.Go),
			wantIDs:   []string{"project-router", "go-style", "web-style", "code-minimalism"},
		},
		{
			name:      "dart-only project keeps dart-style and the ungated skills",
			languages: detect.NewSet(language.Dart),
			wantIDs:   []string{"project-router", "dart-style", "code-minimalism"},
		},
		{
			name:      "typescript-only project keeps both typescript-tagged skills",
			languages: detect.NewSet(language.TypeScript),
			wantIDs:   []string{"project-router", "ts-style", "web-style", "code-minimalism"},
		},
		{
			name:      "a go+dart monorepo selects both baselines",
			languages: detect.NewSet(language.Go, language.Dart),
			wantIDs: []string{
				"project-router", "go-style", "dart-style", "web-style", "code-minimalism",
			},
		},
		{
			name:      "a language with no tagged skill leaves only the ungated ones",
			languages: detect.NewSet(language.Rust),
			wantIDs:   []string{"project-router", "code-minimalism"},
		},
		{
			name:      "no detected language keeps every baseline skill",
			languages: detect.NewSet(),
			wantIDs:   every,
		},
		{
			name:      "a zero-value set is unknown too, not no-languages",
			languages: detect.Set{},
			wantIDs:   every,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sel := Choice{Languages: testCase.languages}
			gotIDs, _ := sel.Resolve(idx)
			if !slices.Equal(gotIDs, testCase.wantIDs) {
				t.Errorf("ids = %v, want %v", gotIDs, testCase.wantIDs)
			}
		})
	}
}
