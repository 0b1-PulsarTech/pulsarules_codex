package mark

import (
	"slices"
	"testing"
)

func TestInlineSpans(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		line  string
		base  int
		ticks tickEscaping
		want  []Region
	}{
		{"a plain pair", "a `b` c", 0, literalTicks, []Region{{2, 5}}},
		{"no ticks", "abc", 0, literalTicks, nil},
		{"a lone run pairs with nothing", "a ` b", 0, literalTicks, nil},
		{"distinct lengths never pair", "` `` ``` ````", 0, literalTicks, nil},
		{"offsets follow the base", "`b`", 10, literalTicks, []Region{{10, 13}}},
		{"an escaped tick pairs the later two", "\\`a `b`", 0, escapedTicks, []Region{{4, 7}}},
		{"an escaped tick pairs when literal", "\\`a `b`", 0, literalTicks, []Region{{1, 5}}},
		{"an escaped double run keeps a tick", "\\``a`", 0, escapedTicks, []Region{{2, 5}}},
		{"two backslashes escape nothing", "\\\\`a`", 0, escapedTicks, []Region{{2, 5}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := inlineSpans(testCase.line, testCase.base, testCase.ticks)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("inlineSpans(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}
