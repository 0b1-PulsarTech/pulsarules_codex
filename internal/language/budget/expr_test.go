package budget

import (
	"strings"
	"testing"
)

func closedAt(end int) Scan { return Scan{End: end, Closed: true} }

func reachedEnd(end int) Scan { return Scan{End: end, Exhausted: true} }

func TestBraceEnd(t *testing.T) {
	t.Parallel()

	deepClosed := strings.Repeat("[", MaxNesting+5) + strings.Repeat("]", MaxNesting+5) + "}x"

	testCases := []struct {
		name string
		text string
		from int
		want Scan
	}{
		{name: "empty body", text: "}x", want: closedAt(1)},
		{name: "simple", text: "a}b", want: closedAt(2)},
		{name: "nested", text: "a{b}c}d", want: closedAt(6)},
		{name: "string brace", text: `"}"}x`, want: closedAt(4)},
		{name: "template brace", text: "`}`}x", want: closedAt(4)},
		{name: "template substitution", text: "`${a}}`}x", want: closedAt(8)},
		{name: "regex with a quote", text: `x.replace(/'/g, "")}y`, want: closedAt(20)},
		{name: "regex with a brace", text: "/}/.test(a)}y", want: closedAt(12)},
		{name: "division is not a regex", text: "a / b / c}y", want: closedAt(10)},
		{name: "post-increment then a slash", text: "x++ / 2}\ny", want: closedAt(8)},
		{name: "post-decrement then a slash", text: "x-- / 2}\ny", want: closedAt(8)},
		{name: "plus then a regex", text: "a + /}/.test(b)}y", want: closedAt(16)},
		{name: "comment brace", text: "a /* } */}y", want: closedAt(10)},
		{name: "comparison", text: "a < b}y", want: closedAt(6)},
		{name: "from the middle", text: "ab c}d", from: 2, want: closedAt(5)},
		{name: "line ends a string", text: "'a\n}b", want: closedAt(4)},
		{name: "unterminated", text: "a", want: reachedEnd(1)},
		{name: "unterminated string", text: `"a`, want: reachedEnd(2)},
		{name: "unterminated template", text: "`a}", want: reachedEnd(3)},
		{name: "unterminated regex", text: "/a}", want: reachedEnd(3)},
		{name: "escape at the end", text: `a\`, want: reachedEnd(2)},
		{name: "misread regex resumes", text: "a = b ? c : /d}\ne", want: reachedEnd(15)},
		{name: "regex on the first line", text: "a ? /b : c\n d\n e", want: reachedEnd(10)},
		{name: "stray closer", text: "a)}", want: Scan{End: 2}},
		{name: "stray closer before a regex", text: "a) /b/}", want: Scan{End: 2}},
		{name: "mismatched closer", text: "(a]}", want: Scan{End: 3}},
		{name: "too deep", text: strings.Repeat("[", MaxNesting+5), want: Scan{End: MaxNesting}},
		{name: "closed after the depth limit", text: deepClosed, want: Scan{End: MaxNesting}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := BraceEnd(testCase.text, testCase.from); got != testCase.want {
				t.Fatalf("BraceEnd %+v, want %+v", got, testCase.want)
			}
		})
	}
}
