package budget

import "testing"

type closeCase struct {
	name      string
	stack     string
	closer    byte
	want      string
	faulted   bool
	wantRegex bool
}

func closeCases() []closeCase {
	return []closeCase{
		{name: "empty stack", closer: ')', faulted: true},
		{name: "matching paren", stack: "{(", closer: ')', want: "{"},
		{name: "arrow paren is a paren", stack: "{a", closer: ')', want: "{"},
		{name: "head paren allows a regex", stack: "{h", closer: ')', want: "{", wantRegex: true},
		{name: "arrow paren is no brace", stack: "{a", closer: '}', want: "{a", faulted: true},
		{name: "other kind faults", stack: "{(", closer: '}', want: "{(", faulted: true},
		{name: "block allows a regex", stack: "(h{", closer: '}', want: "(h", wantRegex: true},
		{name: "object forbids a regex", stack: "(ho", closer: '}', want: "(h"},
		{name: "bracket", stack: "[", closer: ']'},
		{name: "bracket needs its own", stack: "[", closer: ')', want: "[", faulted: true},
		{name: "angles are dropped first", stack: "{(<<", closer: ')', want: "{"},
		{name: "angles alone fault", stack: "<<", closer: '}', faulted: true},
	}
}

func TestCloseBracket(t *testing.T) {
	t.Parallel()

	for _, testCase := range closeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := newScanner(string(testCase.closer))
			for _, kind := range []byte(testCase.stack) {
				scan.push(kind)
			}
			scan.closeBracket(testCase.closer)
			if got := string(scan.stack); got != testCase.want {
				t.Fatalf("stack %q, want %q", got, testCase.want)
			}
			if faulted := scan.fault != ""; faulted != testCase.faulted {
				t.Fatalf("fault %q, want fault %v", scan.fault, testCase.faulted)
			}
			if !testCase.faulted && scan.regexOK != testCase.wantRegex {
				t.Fatalf("regexOK %v, want %v", scan.regexOK, testCase.wantRegex)
			}
		})
	}
}

func TestSpeculativeCount(t *testing.T) {
	t.Parallel()

	scan := newScanner("")
	scan.push(kindArrow)
	scan.push(kindArrow)
	scan.drop()
	scan.push(kindArrow)
	if scan.speculative != 2 || scan.widest != 2 {
		t.Fatalf("speculative %d widest %d, want 2 and 2", scan.speculative, scan.widest)
	}
	scan.drop()
	scan.drop()
	if scan.speculative != 0 || scan.widest != 2 || scan.deepest != 2 {
		t.Fatalf("counts %d %d %d", scan.speculative, scan.widest, scan.deepest)
	}
}

func TestOperatorPostfix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		pos  int
		want bool
	}{
		{name: "post-increment", text: "a++", pos: 2},
		{name: "post-decrement", text: "a--", pos: 2},
		{name: "single plus", text: "a+", pos: 1, want: true},
		{name: "plus after minus", text: "a-+", pos: 2, want: true},
		{name: "minus after plus", text: "a+-", pos: 2, want: true},
		{name: "plus at the start", text: "+", pos: 0, want: true},
		{name: "separated pluses", text: "a + +", pos: 4, want: true},
		{name: "star", text: "a**", pos: 2, want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := newScanner(testCase.text)
			scan.pos = testCase.pos
			scan.operator(testCase.text[testCase.pos])
			if scan.regexOK != testCase.want {
				t.Fatalf("regexOK %v, want %v", scan.regexOK, testCase.want)
			}
		})
	}
}
