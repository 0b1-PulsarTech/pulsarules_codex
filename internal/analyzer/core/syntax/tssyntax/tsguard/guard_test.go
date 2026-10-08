package tsguard

import (
	"fmt"
	"strings"
	"testing"
)

const bracketReason = "skipped: 32769 brackets is over the 32768 parse limit"

type checkCase struct {
	name       string
	text       string
	wantReason string
}

func runCheck(t *testing.T, testCases []checkCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Check(testCase.text); got != testCase.wantReason {
				t.Fatalf("reason %q, want %q", got, testCase.wantReason)
			}
		})
	}
}

func TestCheckLimits(t *testing.T) {
	t.Parallel()

	const nestingReason = "skipped: nesting depth 501 is over the 500 parse limit"
	atLimit := strings.Repeat("(", MaxNesting) + strings.Repeat(")", MaxNesting)
	overLimit := strings.Repeat("(", MaxNesting+1) + strings.Repeat(")", MaxNesting+1)
	runCheck(t, []checkCase{
		{name: "empty", text: ""},
		{name: "ordinary code", text: "function f(a) { return [a, {b: (a)}]; }\n"},
		{name: "exactly at the byte limit", text: strings.Repeat(" ", MaxFileBytes)},
		{
			name:       "one byte over the limit",
			text:       strings.Repeat(" ", MaxFileBytes+1),
			wantReason: "skipped: 1048577 bytes is over the 1048576 byte limit",
		},
		{name: "nesting at the limit", text: atLimit},
		{name: "nesting over the limit", text: overLimit, wantReason: nestingReason},
		{
			name:       "unclosed nesting over the limit",
			text:       strings.Repeat("[", MaxNesting+1),
			wantReason: nestingReason,
		},
		{
			name:       "generic nest over the limit",
			text:       "type X = " + strings.Repeat("A<", MaxNesting+1),
			wantReason: nestingReason,
		},
	})
}

func TestCheckSpeculative(t *testing.T) {
	t.Parallel()

	const reason = "skipped: 11 nested arrow-like parentheses is over the 10 parse limit"
	shape := func(opener, closer string, depth int) string {
		return "x = " + strings.Repeat(opener, depth) + "0" + strings.Repeat(closer, depth) + ";\n"
	}
	runCheck(t, []checkCase{
		{name: "async defaults at the limit", text: shape("async(a=", ")", MaxSpeculative)},
		{name: "async defaults", text: shape("async(a=", ")", 22), wantReason: reason},
		{name: "async calls", text: shape("async(", ")", 11), wantReason: reason},
		{name: "async commas", text: shape("async(a,", ")", 11), wantReason: reason},
		{name: "async patterns", text: shape("async({a}=", ")", 11), wantReason: reason},
		{name: "async ternaries", text: shape("async(a=b>c?", ":d)", 11), wantReason: reason},
		{name: "async bodies", text: shape("async(a)=>async(a=", ")", 11), wantReason: reason},
		{name: "async generics", text: shape("async<T>(a=", ")", 11), wantReason: reason},
		{name: "async generic calls", text: shape("async<T>(", ")", 22), wantReason: reason},
		{name: "generic calls", text: shape("f<T>(", ")", 22), wantReason: reason},
		{name: "block comment", text: shape("async /*c*/ (", ")", 22), wantReason: reason},
		{name: "line comment", text: shape("async // c\n (", ")", 22), wantReason: reason},
		{name: "two comments", text: shape("async /*a*/ /*b*/ (", ")", 22), wantReason: reason},
		{name: "comment before a call", text: shape("f /*c*/ (", ")", 22)},
		{name: "defaults", text: shape("(a=", ")", 4000), wantReason: reason},
		{name: "ternary defaults", text: shape("(a=b>c?", ":d)", 4000), wantReason: reason},
		{name: "arrow defaults", text: shape("(a=b=>", ")", 4000), wantReason: reason},
		{name: "object patterns", text: shape("({a}=", ")", 4000), wantReason: reason},
		{name: "array patterns", text: shape("([a]=", ")", 4000), wantReason: reason},
		{name: "generic arrows", text: shape("<T>(a=", ")", 4000), wantReason: reason},
		{name: "comparison defaults", text: shape("(a=0>=(", "))", 4000), wantReason: reason},
		{name: "nested calls are not arrows", text: shape("f(a, g(b, h(", ")))", 100)},
		{
			name: "loops comparing with less-than",
			text: strings.Repeat("for (var i = 0; i < n; i++) {}\n", 10_000),
		},
	})
}

func TestCheckFaultBeforeLimits(t *testing.T) {
	t.Parallel()

	const nesting = "skipped: nesting depth 501 is over the 500 parse limit"
	runCheck(t, []checkCase{
		{
			name:       "open substitution under deep brackets",
			text:       "`${" + strings.Repeat("[", 500) + "x",
			wantReason: nesting,
		},
	})
}

func TestCheckSpeculativeDepths(t *testing.T) {
	t.Parallel()

	const reason = "skipped: 11 nested arrow-like parentheses is over the 10 parse limit"
	var testCases []checkCase
	for depth := MaxSpeculative + 1; depth <= 2*MaxSpeculative+2; depth++ {
		text := strings.Repeat("async(a=", depth) + "0" + strings.Repeat(")", depth) + ";\n"
		testCases = append(testCases, checkCase{
			name:       fmt.Sprintf("async defaults %d deep", depth),
			text:       text,
			wantReason: reason,
		})
	}
	runCheck(t, testCases)
}

func TestCheckHiddenBrackets(t *testing.T) {
	t.Parallel()

	opened := strings.Repeat("(", MaxOpeners+1)
	fakeClosers := "(" + "/* " + strings.Repeat(")", 10) + " */"
	runCheck(t, []checkCase{
		{
			name: "fake closers in comments cannot hide real openers",
			text: strings.Repeat(fakeClosers, MaxNesting),
		},
		{
			name:       "regex literals fool the depth scan but not the bracket count",
			text:       strings.Repeat("(/)/", MaxOpeners+1),
			wantReason: bracketReason,
		},
		{
			name: "brackets in strings and comments do not count as depth",
			text: strings.Repeat("'(((' // ((((\n", MaxNesting/10),
		},
		{name: "openers inside a comment", text: "/*" + opened + "*/", wantReason: bracketReason},
		{name: "openers inside a string", text: "'" + opened + "'", wantReason: bracketReason},
	})
}

func TestCheckLineTerminators(t *testing.T) {
	t.Parallel()

	opened := strings.Repeat("(", MaxOpeners+1)
	runCheck(t, []checkCase{
		{name: "after a CR line comment", text: "// x\r" + opened, wantReason: bracketReason},
		{name: "after an LS line comment", text: "// x\u2028" + opened, wantReason: bracketReason},
		{name: "after a PS line comment", text: "// x\u2029" + opened, wantReason: bracketReason},
		{name: "after a CR-ended string", text: "'x\r" + opened, wantReason: bracketReason},
		{name: "after an LS-ended string", text: "'x\u2028" + opened, wantReason: bracketReason},
	})
}

func TestCheckRegexLiterals(t *testing.T) {
	t.Parallel()

	const reason = "skipped: 11 nested arrow-like parentheses is over the 10 parse limit"
	shape := strings.Repeat("async(a=", 22) + "0" + strings.Repeat(")", 22) + ";\n"
	whatsapp := "export function render(text: string): string {\n" +
		"\treturn escapeHtml(text)\n" +
		"\t\t.replace(/```(.+?)```/g, '<code>$1</code>')\n" +
		"\t\t.replace(/\\*(.+?)\\*/g, '<b>$1</b>')\n" +
		"\t\t.replace(/_(.+?)_/g, '<i>$1</i>');\n}\n"
	inputValue := "const inputValueOf = (html, name) => {\n" +
		"\tconst tag = new RegExp(`<input[^>]*name=\"${name}\"[^>]*>`).exec(html)?.[0] ?? '';\n" +
		"\treturn decodeEntities(/value=\"([^\"]*)\"/.exec(tag)?.[1] ?? '');\n};\n"
	runCheck(t, []checkCase{
		{name: "quote regex hides nothing", text: "y=/'/;" + shape, wantReason: reason},
		{name: "double quote regex", text: "y=/\"/;" + shape, wantReason: reason},
		{name: "backtick regex", text: "y=/`/;" + shape, wantReason: reason},
		{name: "class regex", text: "y=/[/*]/;" + shape, wantReason: reason},
		{name: "closer regex", text: "y=/)/;" + shape, wantReason: reason},
		{name: "after a head paren", text: "if(1)/'/.test(y);" + shape, wantReason: reason},
		{name: "after a block", text: "{}/'/.test(y);" + shape, wantReason: reason},
		{name: "division before a quote opens a string", text: "y=a/'/;" + shape},
		{name: "keyword property divides", text: "y=a.of/'/;" + shape},
		{name: "spread keyword is a regex", text: "y=[...of/'/];" + shape, wantReason: reason},
		{name: "whatsapp markup", text: whatsapp},
		{name: "input value lookup", text: inputValue},
		{name: "ordinary code", text: "const half = total / 2; const rate = a / b / c;\n"},
	})
}

func TestCheckFaultsDoNotSkip(t *testing.T) {
	t.Parallel()

	runCheck(t, []checkCase{
		{name: "unterminated block comment", text: "x /* y"},
		{name: "unterminated string", text: "x = 'y"},
		{name: "unterminated template", text: "x = `y"},
		{name: "open substitution", text: "x = `${y"},
		{name: "unterminated regex", text: "x = /y"},
		{name: "stray closer", text: "x = y);"},
		{name: "mismatched closer", text: "f(a];"},
		{name: "string ended by a line", text: "x = 'y\nz = 1;\n"},
		{name: "line comment at the end", text: "x = 1; // y"},
	})
}
