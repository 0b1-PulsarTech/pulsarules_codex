package lexical

import (
	"slices"
	"testing"
)

func TestLiterals_TaggedDelimiter(t *testing.T) {
	t.Parallel()

	comments := CommentSyntax{Line: []string{"//"}}
	strs := StringSyntax{Delims: []StringDelim{
		{Open: "`", Close: "`", Multiline: true},
		{Open: "`", Close: "`", Multiline: true, Raw: true, Tag: "raw"},
	}}
	testCases := []struct {
		name string
		src  string
		want []bool
	}{
		{"untagged", "a `x`", []bool{false}},
		{"tagged", "a raw`x`", []bool{true}},
		{"tagged after a blank", "a raw `x`", []bool{true}},
		{"tagged at the start", "raw`x`", []bool{true}},
		{"tagged after a punctuation byte", "(raw`x`)", []bool{true}},
		{"tag inside a longer word", "a myraw`x`", []bool{false}},
		{"tag as a prefix of a longer word", "a rawx`x`", []bool{false}},
		{"tag spelled in a comment", "raw // raw\n`x`", []bool{true}},
		{"tagged then untagged", "raw`x` `y`", []bool{true, false}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got []bool
			for _, span := range Literals(testCase.src, comments, strs).Spans {
				got = append(got, span.Raw)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("raw flags = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}
