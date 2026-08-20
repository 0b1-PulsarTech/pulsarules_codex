package textmarkers

import "testing"

// TestCodeRegions covers what markdown renders as code, since a typographic
// character inside one of these ranges is content being shown, not prose.
// Every marker is written as a Go escape so this fixture never trips the very
// check it exercises.
func TestCodeRegions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		src     string
		inside  []int
		outside []int
	}{
		{
			name:    "prose is never code",
			src:     "a \u2014 b\n",
			outside: []int{2},
		},
		{
			name:   "fenced block hides its content",
			src:    "text\n```\na \u2014 b\n```\nmore\n",
			inside: []int{11},
		},
		{
			name:   "tilde fence works the same",
			src:    "~~~\na \u2014 b\n~~~\n",
			inside: []int{6},
		},
		{
			name:   "an unclosed fence runs to the end",
			src:    "```\na \u2014 b\n",
			inside: []int{6},
		},
		{
			name:   "inline span hides its content",
			src:    "see `a \u2014 b` here\n",
			inside: []int{7},
		},
		{
			name:    "an unclosed backtick is not a span",
			src:     "see ` a \u2014 b\n",
			outside: []int{8},
		},
		{
			name:    "prose after a closed fence is reported again",
			src:     "```\ncode\n```\na \u2014 b\n",
			outside: []int{15},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			regions := codeRegions(testCase.src)
			for _, offset := range testCase.inside {
				if !inCode(regions, offset) {
					t.Errorf("offset %d reads as prose, want code (regions %+v)", offset, regions)
				}
			}
			for _, offset := range testCase.outside {
				if inCode(regions, offset) {
					t.Errorf("offset %d reads as code, want prose (regions %+v)", offset, regions)
				}
			}
		})
	}
}
