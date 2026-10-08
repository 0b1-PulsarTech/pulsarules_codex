package tsparse

import "testing"

func TestLinesOf(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		text   string
		offset int
		want   int
	}{
		{name: "empty text", text: "", offset: 0, want: 1},
		{name: "first byte", text: "ab\ncd", offset: 0, want: 1},
		{name: "the newline itself is on its line", text: "ab\ncd", offset: 2, want: 1},
		{name: "after newline", text: "ab\ncd", offset: 3, want: 2},
		{name: "carriage return is not a break", text: "a\rb\rc", offset: 4, want: 1},
		{name: "line separator is not a break", text: "a\u2028b\nc", offset: 4, want: 1},
		{name: "after the break past a line separator", text: "a\u2028b\nc", offset: 6, want: 2},
		{name: "crlf counts once", text: "a\r\nb\r\nc", offset: 6, want: 3},
		{name: "offset at end", text: "a\n\n\n", offset: 4, want: 4},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := NewLines(testCase.text).Of(testCase.offset); got != testCase.want {
				t.Fatalf("line of %d = %d, want %d", testCase.offset, got, testCase.want)
			}
		})
	}
}
