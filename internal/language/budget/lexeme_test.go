package budget

import "testing"

func TestLineEnd(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		from int
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "no terminator", text: "abc", want: 3},
		{name: "LF", text: "ab\ncd", want: 2},
		{name: "CR", text: "ab\rcd", want: 2},
		{name: "LS", text: "ab\u2028cd", want: 2},
		{name: "PS", text: "ab\u2029cd", want: 2},
		{name: "starts after from", text: "a\nb\nc", from: 2, want: 3},
		{name: "truncated separator", text: "a\xe2\x80", want: 3},
		{name: "other U+202x", text: "a\u202ab", want: 5},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := lineEnd(testCase.text, testCase.from); got != testCase.want {
				t.Fatalf("lineEnd %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestEscapeWidth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want int
	}{
		{name: "plain escape", text: "\\n", want: 2},
		{name: "CR alone", text: "\\\rx", want: 2},
		{name: "CRLF continuation", text: "\\\r\nx", want: 3},
		{name: "at the end", text: "\\", want: 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := escapeWidth(testCase.text, 0); got != testCase.want {
				t.Fatalf("escapeWidth %d, want %d", got, testCase.want)
			}
		})
	}
}
