package knowledge

import (
	"slices"
	"testing"
)

func mustIndex(t *testing.T, skills ...Skill) *Index {
	t.Helper()
	idx, err := NewIndex(skills)
	if err != nil {
		t.Fatalf("NewIndex: %v", err)
	}
	return idx
}

func TestNewIndex_DuplicateSkill(t *testing.T) {
	t.Parallel()

	if _, err := NewIndex([]Skill{{ID: "a"}, {ID: "a"}}); err == nil {
		t.Fatal("NewIndex with a duplicate skill id: err = nil, want an error")
	}
}

func TestSkillsOrdered_ByOrderThenID(t *testing.T) {
	t.Parallel()

	idx := mustIndex(t,
		Skill{ID: "b", Order: 10},
		Skill{ID: "router", Order: 0},
		Skill{ID: "a", Order: 10},
		Skill{ID: "c", Order: 5},
	)

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

	idx := mustIndex(t,
		Skill{ID: "go-style", Languages: []string{"go"}},
		Skill{ID: "dart-style", Languages: []string{"dart"}},
		Skill{ID: "ts-errors", Languages: []string{"typescript", "javascript", "svelte"}},
		Skill{ID: "code-minimalism"},
	)
	idx.Router = RouterSpec{Baseline: RouterBaseline{Always: []RouterBaselineEntry{
		{Skill: "go-style"},
		{Skill: "dart-style"},
		{Skill: "ts-errors"},
		{Skill: "code-minimalism"},
		{Skill: "unlisted"},
	}}}
	testCases := []struct {
		name      string
		index     *Index
		languages []string
		want      []string
	}{
		{"go", idx, []string{"go"}, []string{"go-style", "code-minimalism"}},
		{"dart", idx, []string{"dart"}, []string{"dart-style", "code-minimalism"}},
		{"svelte", idx, []string{"svelte"}, []string{"ts-errors", "code-minimalism"}},
		{"two languages", idx, []string{"go", "dart"}, []string{
			"go-style", "dart-style", "code-minimalism",
		}},
		{"no language keeps the neutral ones", idx, nil, []string{"code-minimalism"}},
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

func TestFits(t *testing.T) {
	t.Parallel()

	idx := mustIndex(t,
		Skill{ID: "go-style", Languages: []string{"go"}},
		Skill{ID: "code-minimalism"},
	)
	testCases := []struct {
		name      string
		id        string
		languages []string
		want      bool
	}{
		{"language match", "go-style", []string{"go"}, true},
		{"language mismatch", "go-style", []string{"dart"}, false},
		{"neutral skill fits any project", "code-minimalism", []string{"dart"}, true},
		{"neutral skill fits no language", "code-minimalism", nil, true},
		{"unknown id fits nothing", "unlisted", []string{"go"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := idx.Fits(testCase.id, testCase.languages...); got != testCase.want {
				t.Errorf("Fits(%q, %v) = %v, want %v",
					testCase.id, testCase.languages, got, testCase.want)
			}
		})
	}
}
