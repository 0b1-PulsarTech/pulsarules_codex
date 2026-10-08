package tsguard

import (
	"strings"
	"testing"
)

func TestCountOpeners(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: ""},
		{name: "closers do not count", text: ")]}>", want: 0},
		{name: "every opener kind", text: "([{<", want: 4},
		{name: "strings and comments count", text: "'(' // [ /* { */ <", want: 4},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := countOpeners(testCase.text); got != testCase.want {
				t.Fatalf("openers %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestCountOperators(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: ""},
		{name: "names and brackets do not count", text: "a1_$ ([{ }]) ;.\n", want: 0},
		{name: "every operator byte", text: "!~=?:*+-&|^%>,", want: 14},
		{name: "strings and comments count", text: "'!' // = ?", want: 3},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := countOperators(testCase.text); got != testCase.want {
				t.Fatalf("operators %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestCheckGauges(t *testing.T) {
	t.Parallel()

	const over = "skipped: 65537 operators is over the 65536 parse limit"
	rep := strings.Repeat
	half := MaxOperators / 2
	testCases := []struct {
		name string
		text string
		want string
	}{
		{name: "ordinary code", text: "const total = (a, b) => a + b;\n"},
		{name: "openers at the limit", text: rep("(", MaxOpeners)},
		{name: "openers over the limit", text: rep("[", MaxOpeners+1), want: bracketReason},
		{name: "operators at the limit", text: rep("!", MaxOperators) + "a"},
		{name: "prefix chain", text: "x=" + rep("!", MaxOperators) + "a", want: over},
		{name: "unary pair chain", text: "x=" + rep("~!", half) + "a", want: over},
		{name: "arrow chain", text: "f=" + rep("a=>", half) + "a", want: over},
		{name: "assignment chain", text: "x" + rep("=a", MaxOperators+1), want: over},
		{name: "conditional chain", text: "x=" + rep("a?a:", half), want: over},
		{name: "power chain", text: "x=a" + rep("**a", half), want: over},
		{name: "comma list", text: "x=(a" + rep(",a", MaxOperators) + ")", want: over},
		{name: "unary words are bounded by bytes", text: "x=" + rep("void ", 50_000) + "a"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := checkGauges(testCase.text); got != testCase.want {
				t.Fatalf("reason %q, want %q", got, testCase.want)
			}
		})
	}
}
