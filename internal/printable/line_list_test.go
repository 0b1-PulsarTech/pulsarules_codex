package printable

import "testing"

func TestLineList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		lines     []int
		limit     int
		wantLen   int
		wantNoun  string
		wantExtra int
		wantHead  string
	}{
		{"empty", nil, 5, 0, "line", 0, ""},
		{"one", []int{3}, 5, 1, "line", 0, "3"},
		{"consecutive duplicates collapse", []int{3, 3, 3}, 5, 1, "line", 0, "3"},
		{"non-adjacent duplicate kept", []int{3, 4, 3}, 5, 3, "lines", 0, "3, 4, 3"},
		{"exactly the limit", []int{1, 2, 3}, 3, 3, "lines", 0, "1, 2, 3"},
		{"over the limit", []int{1, 2, 3, 4, 5}, 3, 5, "lines", 2, "1, 2, 3"},
		{"zero limit", []int{1, 2}, 0, 2, "lines", 2, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var list LineList
			for _, line := range testCase.lines {
				list.Add(line)
			}
			if got := list.Len(); got != testCase.wantLen {
				t.Errorf("Len = %d, want %d", got, testCase.wantLen)
			}
			if got := list.Noun(); got != testCase.wantNoun {
				t.Errorf("Noun = %q, want %q", got, testCase.wantNoun)
			}
			if got := list.Extra(testCase.limit); got != testCase.wantExtra {
				t.Errorf("Extra = %d, want %d", got, testCase.wantExtra)
			}
			if got := list.Head(testCase.limit); got != testCase.wantHead {
				t.Errorf("Head = %q, want %q", got, testCase.wantHead)
			}
		})
	}
}
