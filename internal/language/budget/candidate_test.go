package budget

import (
	"slices"
	"strings"
	"testing"
)

type speculativeCase struct {
	name string
	text string
	want bool
}

func speculativeCases() []speculativeCase {
	return slices.Concat(callCases(), parameterCases())
}

func callCases() []speculativeCase {
	return []speculativeCase{
		{name: "async call", text: "x = async(", want: true},
		{name: "async after blanks", text: "x = async \n\t(", want: true},
		{name: "longer word ending in async", text: "x = myasync(a, b)"},
		{name: "plain call", text: "f(a, b)"},
		{name: "generic call", text: "f<T>(0)", want: true},
		{name: "async generic call", text: "x = async<T>(0)", want: true},
		{name: "arrow body", text: "f = a => (0)"},
		{name: "arrow body without blanks", text: "f = a=>(0)"},
		{name: "member call", text: "a.b(c, d)"},
		{name: "call of a call", text: "f()(a, b)"},
		{name: "call of an index", text: "a[0](b, c)"},
		{name: "default value", text: "x = (a = 1)", want: true},
		{name: "parameter list", text: "x = (a, b)", want: true},
		{name: "single name", text: "x = (a)", want: true},
		{name: "optional name", text: "x = (a?)", want: true},
		{name: "annotated name", text: "x = (a: T)", want: true},
		{name: "blanks before the name", text: "x = (\n\ta = 1)", want: true},
	}
}

func parameterCases() []speculativeCase {
	return []speculativeCase{
		{name: "object pattern", text: "x = ({a})", want: true},
		{name: "array pattern", text: "x = ([a])", want: true},
		{name: "rest", text: "x = (...a)", want: true},
		{name: "after return", text: "return (a, b)", want: true},
		{name: "after return without a space", text: "return(a=1)", want: true},
		{name: "after typeof", text: "typeof (a)", want: true},
		{name: "after a type argument list", text: "x = f<T>(a, b)", want: true},
		{name: "empty list", text: "x = ()"},
		{name: "arithmetic", text: "x = (a + b)"},
		{name: "member access", text: "x = (a.b)"},
		{name: "number", text: "x = (1)"},
		{name: "two dots", text: "x = (..a)"},
		{name: "two names", text: "x = (a b)"},
		{name: "name at the end", text: "x = (a"},
		{name: "nothing after", text: "x = ("},
		{name: "at the start", text: "(a, b)", want: true},
		{name: "string before", text: "'s'(a, b)", want: true},
	}
}

func TestSpeculative(t *testing.T) {
	t.Parallel()

	testCases := speculativeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pos := strings.LastIndex(testCase.text, "(")
			if got := speculative(testCase.text, pos, pos); got != testCase.want {
				t.Fatalf("speculative %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestWordBefore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		prefix   string
		wantWord string
		wantLast byte
	}{
		{name: "empty"},
		{name: "blanks only", prefix: " \n"},
		{name: "word", prefix: "x = async", wantWord: "async", wantLast: 'c'},
		{name: "word after blanks", prefix: "return \n", wantWord: "return", wantLast: 'n'},
		{name: "dollar and digits", prefix: "+ $a1_", wantWord: "$a1_", wantLast: '_'},
		{name: "punctuation", prefix: "a)", wantLast: ')'},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			word, last := wordBefore(testCase.prefix)
			if word != testCase.wantWord || last != testCase.wantLast {
				t.Fatalf("got %q %q, want %q %q", word, last, testCase.wantWord, testCase.wantLast)
			}
		})
	}
}
