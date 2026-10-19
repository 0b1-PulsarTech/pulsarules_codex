package install

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrompter_Ask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trims the line", input: "  1,4 \nrest\n", want: "1,4"},
		{name: "blank line", input: "\n", want: ""},
		{name: "eof without newline", input: "a", want: "a"},
		{name: "empty input", input: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			console := newPrompter(bytes.NewReader([]byte(testCase.input)), &out)
			if got := console.ask("skills> "); got != testCase.want {
				t.Errorf("answer = %q, want %q", got, testCase.want)
			}
			if out.String() != "skills> " {
				t.Errorf("question written = %q, want %q", out.String(), "skills> ")
			}
		})
	}
}

func TestPrompter_AskSharesBuffer(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	console := newPrompter(strings.NewReader("first\nsecond\n"), &out)
	if got := console.ask("a> "); got != "first" {
		t.Fatalf("first answer = %q", got)
	}
	if got := console.ask("b> "); got != "second" {
		t.Fatalf("second answer = %q", got)
	}
}
