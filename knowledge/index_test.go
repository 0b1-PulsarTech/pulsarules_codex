package knowledge

import "testing"

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
