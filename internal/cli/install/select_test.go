package install

import (
	"slices"
	"testing"
)

// TestParseSkillSelection covers ranges, comma lists, "a" for all, blank for
// none, and every rejected shape (out of range, zero/negative, garbage, an
// inverted range).
func TestParseSkillSelection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		total   int
		want    []int
		wantErr bool
	}{
		{name: "blank selects nothing", input: "", total: 5, want: nil},
		{name: "blank with surrounding newline", input: "\n", total: 5, want: nil},
		{name: "a selects all", input: "a", total: 3, want: []int{1, 2, 3}},
		{name: "A is case-insensitive", input: "A", total: 2, want: []int{1, 2}},
		{name: "single index", input: "4", total: 5, want: []int{4}},
		{name: "comma list", input: "1,4", total: 5, want: []int{1, 4}},
		{name: "range", input: "7-9", total: 9, want: []int{7, 8, 9}},
		{name: "mixed list and range", input: "1,4,7-9", total: 9, want: []int{1, 4, 7, 8, 9}},
		{name: "deduplicates overlapping picks", input: "1,1,1-2", total: 5, want: []int{1, 2}},
		{name: "tolerates spaces", input: " 1 , 4 , 7 - 9 ", total: 9, want: []int{1, 4, 7, 8, 9}},
		{name: "zero is out of range", input: "0", total: 5, wantErr: true},
		{name: "index above total is out of range", input: "6", total: 5, wantErr: true},
		{name: "inverted range is rejected", input: "9-7", total: 9, wantErr: true},
		{name: "garbage token is rejected", input: "x", total: 5, wantErr: true},
		{name: "garbage range start is rejected", input: "x-3", total: 5, wantErr: true},
		{name: "garbage range end is rejected", input: "1-x", total: 5, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSkillSelection(testCase.input, testCase.total)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
