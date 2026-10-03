package historynarration

import (
	"slices"
	"testing"
)

func TestGoSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		ok     bool
		expect []segment
	}{
		{"line comment", "package p\n// a\n", true, []segment{{2, " a"}}},
		{
			"block comment keeps each line",
			"package p\n/* a\nb */\n",
			true,
			[]segment{{2, " a"}, {3, "b "}},
		},
		{"string is not a comment", "package p\nvar s = \"// a\"\n", true, nil},
		{"parse failure", "package p\nfunc (", false, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := goSegments([]byte(testCase.source))
			if ok != testCase.ok || !slices.Equal(got, testCase.expect) {
				t.Fatalf("goSegments = %v, %v; want %v, %v", got, ok, testCase.expect, testCase.ok)
			}
		})
	}
}
