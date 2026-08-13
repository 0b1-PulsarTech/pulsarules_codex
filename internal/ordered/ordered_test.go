package ordered

import (
	"slices"
	"testing"
)

func TestDedupe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "nil input", input: nil, want: nil},
		{name: "empty input", input: []string{}, want: nil},
		{name: "no duplicates", input: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "adjacent duplicates", input: []string{"a", "a", "b"}, want: []string{"a", "b"}},
		{
			name:  "distant duplicates keep order",
			input: []string{"b", "a", "b"},
			want:  []string{"b", "a"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Dedupe(testCase.input)
			if !slices.Equal(got, testCase.want) || (got == nil) != (testCase.want == nil) {
				t.Errorf("Dedupe(%v) = %#v, want %#v", testCase.input, got, testCase.want)
			}
		})
	}
}
