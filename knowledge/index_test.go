package knowledge

import "testing"

// TestSkillsOrdered asserts skills sort by Order then ID, so the router and
// listings present them in composition order regardless of declaration order.
func TestSkillsOrdered(t *testing.T) {
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
	for i, id := range want {
		if ordered[i].ID != id {
			t.Errorf("position %d = %q, want %q", i, ordered[i].ID, id)
		}
	}
}
