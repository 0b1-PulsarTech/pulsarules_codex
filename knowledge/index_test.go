package knowledge

import (
	"slices"
	"testing"
)

func TestSkillsOrdered_ByOrderThenID(t *testing.T) {
	t.Parallel()

	idx := &Index{Skills: []Skill{
		{ID: "b", Order: 10},
		{ID: "router", Order: 0},
		{ID: "a", Order: 10},
		{ID: "c", Order: 5},
	}}

	want := []string{"router", "c", "a", "b"}
	ordered := idx.SkillsOrdered()
	if len(ordered) != len(want) {
		t.Fatalf("len = %d, want %d", len(ordered), len(want))
	}
	for pos, id := range want {
		if ordered[pos].ID != id {
			t.Errorf("position %d = %q, want %q", pos, ordered[pos].ID, id)
		}
	}
}

func TestBaselineFor(t *testing.T) {
	t.Parallel()

	idx := &Index{
		Skills: []Skill{
			{ID: "go-style", Languages: []string{"go"}},
			{ID: "dart-style", Languages: []string{"dart"}},
			{ID: "ts-errors", Languages: []string{"typescript", "javascript", "svelte"}},
			{ID: "code-minimalism"},
		},
		Router: RouterSpec{Baseline: RouterBaseline{Always: []RouterBaselineEntry{
			{Skill: "go-style"},
			{Skill: "dart-style"},
			{Skill: "ts-errors"},
			{Skill: "code-minimalism"},
			{Skill: "unlisted"},
		}}},
	}
	testCases := []struct {
		name      string
		index     *Index
		languages []string
		want      []string
	}{
		{"go", idx, []string{"go"}, []string{"go-style", "code-minimalism", "unlisted"}},
		{"dart", idx, []string{"dart"}, []string{"dart-style", "code-minimalism", "unlisted"}},
		{"svelte", idx, []string{"svelte"}, []string{"ts-errors", "code-minimalism", "unlisted"}},
		{"two languages", idx, []string{"go", "dart"}, []string{
			"go-style", "dart-style", "code-minimalism", "unlisted",
		}},
		{"no language keeps the neutral ones", idx, nil, []string{"code-minimalism", "unlisted"}},
		{"nil index", nil, []string{"go"}, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := testCase.index.BaselineFor(testCase.languages...)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("BaselineFor(%v) = %v, want %v", testCase.languages, got, testCase.want)
			}
		})
	}
}
