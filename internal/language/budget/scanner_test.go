package budget

import "testing"

type runCase struct {
	name        string
	text        string
	wantDeepest int
	wantWidest  int
	wantFault   string
}

func checkRun(t *testing.T, testCases []runCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := newScanner(testCase.text)
			if got := scan.run(); got != testCase.wantDeepest {
				t.Fatalf("deepest %d, want %d", got, testCase.wantDeepest)
			}
			if scan.widest != testCase.wantWidest {
				t.Fatalf("widest %d, want %d", scan.widest, testCase.wantWidest)
			}
			if scan.fault != testCase.wantFault {
				t.Fatalf("fault %q, want %q", scan.fault, testCase.wantFault)
			}
		})
	}
}

func TestScannerRunBrackets(t *testing.T) {
	t.Parallel()

	const unmatched = "a closing bracket without its opener"

	checkRun(t, []runCase{
		{name: "empty", text: ""},
		{name: "balanced", text: "((()))", wantDeepest: 3},
		{name: "mixed brackets", text: "{[(", wantDeepest: 3},
		{name: "siblings", text: "()()()", wantDeepest: 1},
		{name: "stray closer faults", text: ")))((", wantFault: unmatched},
		{name: "generics", text: "A<B<C>>", wantDeepest: 2},
		{name: "arrow is not a closer", text: "(a) => (b)", wantDeepest: 1, wantWidest: 1},
		{name: "arrow keeps a paren", text: "(x => (((y))))", wantDeepest: 4, wantWidest: 2},
		{name: "arrow keeps an angle", text: "A<() => B<(((x)))>>", wantDeepest: 5, wantWidest: 1},
		{name: "comparison does not close a paren", text: "(a > b)", wantDeepest: 1},
		{name: "greater-than needs its own angle", text: "((a > b))", wantDeepest: 2},
		{name: "mismatched closer faults", text: "(]((", wantDeepest: 1, wantFault: unmatched},
		{
			name:        "closer of another kind faults",
			text:        "{(}}(",
			wantDeepest: 2,
			wantFault:   unmatched,
		},
		{name: "less or equal is not an opener", text: "a <= b << c", wantDeepest: 0},
		{name: "jsx siblings", text: "<a></a><b></b>", wantDeepest: 1},
	})
}

func TestScannerRunComments(t *testing.T) {
	t.Parallel()

	checkRun(t, []runCase{
		{name: "line comment", text: "// (((\n(", wantDeepest: 1},
		{name: "line comment ends at CR", text: "// (((\r(", wantDeepest: 1},
		{name: "line comment ends at CRLF", text: "// (((\r\n(", wantDeepest: 1},
		{name: "line comment ends at LS", text: "// (((\u2028(", wantDeepest: 1},
		{name: "line comment ends at PS", text: "// (((\u2029(", wantDeepest: 1},
		{name: "U+2027 does not end a comment", text: "// (((\u2027(", wantDeepest: 0},
		{name: "truncated LS does not end a comment", text: "// \xe2\x80", wantDeepest: 0},
		{name: "block comment", text: "/* ((( */ (", wantDeepest: 1},
		{
			name: "unterminated block comment", text: "( /* (((",
			wantDeepest: 1, wantFault: "an unterminated comment",
		},
		{name: "division is not a comment", text: "a / b / (c)", wantDeepest: 1, wantWidest: 1},
	})
}

func TestScannerRunStrings(t *testing.T) {
	t.Parallel()

	checkRun(t, []runCase{
		{name: "single quoted", text: `'(((' (`, wantDeepest: 1},
		{name: "double quoted with escape", text: `"\"(((" (`, wantDeepest: 1},
		{name: "unterminated string ends at LF", text: "'((\n(", wantDeepest: 1},
		{name: "unterminated string ends at CR", text: "'((\r(", wantDeepest: 1},
		{name: "unterminated string ends at LS", text: "'((\u2028(", wantDeepest: 1},
		{name: "unterminated string ends at PS", text: "'((\u2029(", wantDeepest: 1},
		{name: "line continuation with CRLF", text: "'a\\\r\n((' (", wantDeepest: 1},
		{name: "line continuation with LF", text: "'a\\\n((' (", wantDeepest: 1},
		{name: "string cut off by the end", text: "'((", wantFault: "an unterminated string"},
	})
}

func TestScannerRunTemplates(t *testing.T) {
	t.Parallel()

	const unterminatedTmpl = "an unterminated template"

	checkRun(t, []runCase{
		{name: "template text", text: "`((` (", wantDeepest: 1},
		{name: "template with escaped backtick", text: "`\\`((` (", wantDeepest: 1},
		{name: "template substitution counts", text: "`${(1)}`", wantDeepest: 2},
		{name: "nested templates", text: "`${`${(1)}`}`", wantDeepest: 3},
		{name: "template resumes after substitution", text: "`${a} (((` (", wantDeepest: 1},
		{name: "open substitution", text: "`abc ${", wantDeepest: 1, wantFault: unterminatedTmpl},
		{name: "unterminated template", text: "`abc", wantFault: unterminatedTmpl},
		{
			name:        "template cut off after a substitution",
			text:        "`${a}",
			wantDeepest: 1,
			wantFault:   unterminatedTmpl,
		},
	})
}

func TestScannerRunRegex(t *testing.T) {
	t.Parallel()

	const unterminated = "an unterminated regular expression"
	checkRun(t, []runCase{
		{name: "regex at the start", text: "/(((/ (", wantDeepest: 1},
		{name: "regex after an operator", text: "x = /(((/ + (", wantDeepest: 1},
		{name: "regex after a call", text: "f(/(((/)", wantDeepest: 1},
		{name: "quote in a regex", text: "x = /'/; (", wantDeepest: 1},
		{name: "double quote in a regex", text: `x = /"/; (`, wantDeepest: 1},
		{name: "backtick in a regex", text: "x = /`/; (", wantDeepest: 1},
		{name: "closer in a regex", text: "x = (/)/; (", wantDeepest: 2},
		{name: "slash in a class", text: "x = /[/*]/; (", wantDeepest: 1},
		{name: "escaped slash", text: `x = /\/(/; (`, wantDeepest: 1},
		{name: "escaped bracket does not open a class", text: `x = /\[/; (`, wantDeepest: 1},
		{name: "flags are not code", text: "x = /a/g; (", wantDeepest: 1},
		{name: "regex after return", text: "return /'/; (", wantDeepest: 1},
		{name: "regex after typeof", text: "typeof /'/; (", wantDeepest: 1},
		{name: "regex after case", text: "case /'/: (", wantDeepest: 1},
		{name: "regex after in", text: "a in /'/; (", wantDeepest: 1},
		{name: "regex after void", text: "void /'/; (", wantDeepest: 1},
		{name: "regex after a head paren", text: "if (a) /'/.test(b); (", wantDeepest: 1},
		{name: "regex after a while paren", text: "while (a) /'/.test(b); (", wantDeepest: 1},
		{name: "regex after a for paren", text: "for (;;) /'/.test(b); (", wantDeepest: 1},
		{name: "regex after a with paren", text: "with (a) /'/.test(b); (", wantDeepest: 1},
		{name: "regex after a block", text: "{} /'/.test(b); (", wantDeepest: 1},
		{name: "division after a name", text: "a / b / (c)", wantDeepest: 1, wantWidest: 1},
		{name: "division after a number", text: "1 / 2 / (c)", wantDeepest: 1, wantWidest: 1},
		{name: "division after a paren", text: "(a) / '/' (", wantDeepest: 1, wantWidest: 1},
		{name: "division after a bracket", text: "a[0] / '/' (", wantDeepest: 1},
		{name: "division after an object", text: "x = {} / '/' (", wantDeepest: 1},
		{name: "division after a return value", text: "return {} / '/' (", wantDeepest: 1},
		{name: "division after a caret value", text: "a ^ {} / '/' (", wantDeepest: 1},
		{name: "regex after a division", text: "a / /'/.test(b); (", wantDeepest: 1},
		{name: "division after a string", text: "'a' / '/' (", wantDeepest: 1},
		{name: "division after a template", text: "`a` / '/' (", wantDeepest: 1},
		{name: "closing tag is no regex", text: "<a>b</a> '(' (", wantDeepest: 1},
		{name: "regex ends at a line end", text: "x = /(((\n(", wantDeepest: 1},
		{name: "regex ends at CR", text: "x = /(((\r(", wantDeepest: 1},
		{name: "regex cut off by the end", text: "x = /((", wantFault: unterminated},
		{name: "regex after a comment", text: "x = /* c */ /'/; (", wantDeepest: 1},
		{name: "template substitution allows a regex", text: "`${/'/}` (", wantDeepest: 1},
	})
}
