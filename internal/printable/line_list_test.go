package printable

import "testing"

func TestLineList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		lines     []int
		limit     int
		wantExtra int
		wantLabel string
	}{
		{"empty", nil, 5, 0, "line "},
		{"one", []int{3}, 5, 0, "line 3"},
		{"consecutive duplicates collapse", []int{3, 3, 3}, 5, 0, "line 3"},
		{"non-adjacent duplicate kept", []int{3, 4, 3}, 5, 0, "lines 3, 4, 3"},
		{"exactly the limit", []int{1, 2, 3}, 3, 0, "lines 1, 2, 3"},
		{"over the limit", []int{1, 2, 3, 4, 5}, 3, 2, "lines 1, 2, 3"},
		{"zero limit", []int{1, 2}, 0, 2, "lines "},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var list LineList
			for _, line := range testCase.lines {
				list.Add(line)
			}
			if got := list.Extra(testCase.limit); got != testCase.wantExtra {
				t.Errorf("Extra = %d, want %d", got, testCase.wantExtra)
			}
			if got := list.Label(testCase.limit); got != testCase.wantLabel {
				t.Errorf("Label = %q, want %q", got, testCase.wantLabel)
			}
		})
	}
}
