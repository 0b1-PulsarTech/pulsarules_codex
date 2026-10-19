package helperbin

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func TestTail(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 500) + "end"
	bullets := "a" + strings.Repeat("\u2022", 200)
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trims space", input: "  hi \n", want: "hi"},
		{name: "keeps the end", input: long, want: long[len(long)-400:]},
		{
			name: "escapes a clipped newline", input: strings.Repeat("x", 500) + "\nend",
			want: strings.Repeat("x", 396) + `\nend`,
		},
		{
			name: "short text keeps raw newlines even when escaping would exceed the clip",
			input: strings.Repeat(
				"e00\n",
				90,
			),
			want: strings.TrimSpace(strings.Repeat("e00\n", 90)),
		},
		{name: "never splits a rune", input: bullets, want: bullets[len(bullets)-399:]},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := tail(testCase.input)
			if got != testCase.want || !utf8.ValidString(got) {
				t.Errorf("tail = %q, want valid UTF-8 %q", got, testCase.want)
			}
		})
	}
}

func TestStepError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		result process.Result
		err    error
		want   string
	}{
		{
			name: "execx error unwraps to its cause",
			err: &process.Error{
				Name: "dart", Args: []string{"x"}, Stderr: "bad", Err: errors.New("boom"),
			},
			result: process.Result{Stderr: "bad"}, want: "dart run: boom: bad",
		},
		{name: "empty output has no trailing colon", want: "dart run: boom"},
		{
			name:   "whitespace-only output has no trailing colon",
			result: process.Result{Stderr: " \n", Stdout: "\t"}, want: "dart run: boom",
		},
		{
			name:   "stderr is appended",
			result: process.Result{Stderr: "bad"}, want: "dart run: boom: bad",
		},
		{
			name:   "stdout alone is appended",
			result: process.Result{Stdout: "out"}, want: "dart run: boom: out",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runErr := testCase.err
			if runErr == nil {
				runErr = errors.New("boom")
			}
			err := StepError("run", testCase.result, runErr)
			if err.Error() != testCase.want {
				t.Errorf("StepError = %q, want %q", err, testCase.want)
			}
		})
	}
}
