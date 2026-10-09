package tsalias

import (
	"strings"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, in, want string
		reject         bool
	}{
		{name: "plain", in: `{"a": 1}`, want: `{"a": 1}`},
		{name: "line comment", in: "{\"a\": 1 // x\n}", want: "{\"a\": 1 \n}"},
		{name: "block comment", in: `{/* x */"a": 1}`, want: `{ "a": 1}`},
		{name: "unclosed block", in: `{"a": 1} /* x`, want: `{"a": 1}  `},
		{name: "slashes in strings", in: `{"u": "http://a/*b*/"}`, want: `{"u": "http://a/*b*/"}`},
		{name: "escaped quote", in: `{"u": "a\"//b"}`, want: `{"u": "a\"//b"}`},
		{name: "trailing comma object", in: "{\"a\": 1,\n}", want: "{\"a\": 1\n}"},
		{name: "trailing comma array", in: `[1, 2, ]`, want: `[1, 2 ]`},
		{
			name: "comma before comment then close",
			in:   `{"a": 1, // x` + "\n}", want: "{\"a\": 1 \n}",
		},
		{name: "comma in string kept", in: `{"a": ",}"}`, want: `{"a": ",}"}`},
		{name: "inner comma kept", in: `[1,2]`, want: `[1,2]`},
		{name: "unterminated string", in: `{"a`, want: `{"a`},
		{name: "over the cap", in: strings.Repeat(" ", maxConfigBytes+1), reject: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := stripJSONC([]byte(testCase.in))
			if ok == testCase.reject || string(got) != testCase.want {
				t.Fatalf("stripJSONC = %q, %v", got, ok)
			}
		})
	}
}
