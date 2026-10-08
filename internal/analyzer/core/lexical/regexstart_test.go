package lexical

import (
	"slices"
	"strings"
	"testing"
)

func regexSyntax() (CommentSyntax, StringSyntax) {
	comments, strs := testSyntax()
	strs.Delims = []StringDelim{
		{Open: `"`, Close: `"`, Escapes: true},
		{Open: `'`, Close: `'`, Escapes: true},
		{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: "${"},
		{Open: "/", Close: "/", Escapes: true, Regex: true},
	}
	strs.RegexAfter = RegexContext{
		After:      "(,=:[!&|?{};+-*%<>~^",
		Words:      []string{"return", "typeof", "case", "in", "of", "void", "yield", "await"},
		ParenWords: []string{"if", "while", "for", "with"},
	}
	return comments, strs
}

func TestLiterals_RegexOpens(t *testing.T) {
	t.Parallel()

	testCases := []literalCase{
		{"after an open paren", "m.matchAll(/<li>[^>]*/g)", []string{`/<li>[^>]*/`}},
		{"after an assignment", `x = /value="([^"]*)"/`, []string{`/value="([^"]*)"/`}},
		{"after a logical and", `a && /class="badge/.test(c)`, []string{`/class="badge/`}},
		{"after a comma", "f(a, /x/)", []string{"/x/"}},
		{"at the start of the file", "/a\"b/.test(c)", []string{`/a"b/`}},
		{"at the start of a line", "a;\n  /a\"b/.test(c)", []string{`/a"b/`}},
		{"at the start of a CRLF line", "a;\r\n /a\"b/.test(c)", []string{`/a"b/`}},
		{"after a comment line", "a;\n// c\n/a\"b/.test(c)", []string{`/a"b/`}},
		{"after a block comment", "a; /* it's */ /a\"b/.test(c)", []string{`/a"b/`}},
		{"after an if header", "if (ok) /\"/.test(s)", []string{`/"/`}},
		{"after a while header", "while (more()) /\"/.test(s)", []string{`/"/`}},
		{"after a for header", "for (;;) /\"/.test(s)", []string{`/"/`}},
		{"after a nested if header", "if (f(a)) /\"/.test(s)", []string{`/"/`}},
		{"after a with header on a new line", "with (o)\n/\"/.test(s)", []string{`/"/`}},
		{"after a negation", `x = !/"/.test(s) + 'q'`, []string{`/"/`, `'q'`}},
		{"after a unary plus", `a + +/q"/.test(s)`, []string{`/q"/`}},
		{"after a binary plus", `ab+/q"/.test(s)`, []string{`/q"/`}},
		{"after a keyword with a blank before the bang", `return !/"/.test(s)`, []string{`/"/`}},
		{"after a block close", "}\n/x/.test(c)", []string{"/x/"}},
		{"after return", `return /x"/.test(c)`, []string{`/x"/`}},
		{"after typeof", `typeof /x/`, []string{"/x/"}},
		{"after await", `await /x/`, []string{"/x/"}},
		{"quotes inside the regex", `s.replace(/"/g, "ok")`, []string{`/"/`, `"ok"`}},
		{"a backtick inside the regex", "x = /`/; y = `t`", []string{"/`/", "`t`"}},
		{"slash inside a class", `x = /[/]"/`, []string{`/[/]"/`}},
		{"escaped slash", `x = /a\/b"/`, []string{`/a\/b"/`}},
		{"escaped bracket", `x = /\[/ + "q"`, []string{`/\[/`, `"q"`}},
		{"after a ternary", "a ? /x/ : /y/", []string{"/x/", "/y/"}},
		{"inside interpolation code", "`${ /x/ }`", []string{"`", "/x/", "`"}},
		{"unterminated is a division", "x = /abc\n'y'", []string{"'y'"}},
		{"unterminated at EOF is a division", "x = /abc", nil},
		{"code after a failed regex is scanned", "x = /ab 'q' + \"r\"\n", []string{"'q'", `"r"`}},
		{"a failed regex blocks later ones", "x = /[ (/a\"/ + 'q'\n", []string{"\"/ + 'q'"}},
		{"the next line may open a regex again", "x = /[\ny = /a\"/", []string{`/a"/`}},
	}
	runRegexCases(t, testCases)
}

func TestLiterals_RegexStaysDivision(t *testing.T) {
	t.Parallel()

	testCases := []literalCase{
		{"after identifiers", "a / b / c", nil},
		{"after numbers", "x = 1 / 2 / 3", nil},
		{"after a close paren", "(a) / b / c", nil},
		{"after a close bracket", "a[0] / b / c", nil},
		{"after a string", `"x" / 2 / 'y'`, []string{`"x"`, `'y'`}},
		{"after a plain word", "total / count / 'q'", []string{"'q'"}},
		{"after a call", "f(x) / 2 + 'q'", []string{"'q'"}},
		{"after a call inside an if body", "if (a) b(c) / 2 + 'q'", []string{"'q'"}},
		{"after a method named if", "o.if(x) / 2 + 'q'", []string{"'q'"}},
		{"after a longer word", "elif (a) / 2 + 'q'", []string{"'q'"}},
		{"after a call, an earlier header", "if (a) {} f(b) / 2 + 'q'", []string{"'q'"}},
		{"after a stray close paren", ") / 2 + 'q'", []string{"'q'"}},
		{"after a postfix increment", "i++ / 2 + 'q'", []string{"'q'"}},
		{"after a postfix decrement", "a[0]-- / 2 + 'q'", []string{"'q'"}},
		{"after a postfix call result", "f()++ / 2 + 'q'", []string{"'q'"}},
		{"after a non-null assertion", "done! / total + 'q'", []string{"'q'"}},
		{"after a non-null paren", "(a)! / 2 + 'q'", []string{"'q'"}},
		{"after a non-null index", "a[0]! / 2 + 'q'", []string{"'q'"}},
		{"keyword as a property", "node.return / 2 + 'q'", []string{"'q'"}},
		{"keyword as an optional property", "a?.typeof / 2 + 'q'", []string{"'q'"}},
		{"line start after a word", "total\n  / count + 'q'", []string{"'q'"}},
		{"line start after a paren", "f(x)\n/ 2 + 'q'", []string{"'q'"}},
		{"line start after a CRLF word", "total\r\n/ count + 'q'", []string{"'q'"}},
		{"line start behind a comment", "total // c\n/ count + 'q'", []string{"'q'"}},
		{"line start behind a block comment", "total /* c */\n/ count + 'q'", []string{"'q'"}},
		{"closing tag", "<b>x</b> 'a'", []string{"'a'"}},
		{"line comment", "x = // \"c\"\n'ok'", []string{"'ok'"}},
		{"block comment", "x = /* 'c' */ 'ok'", []string{"'ok'"}},
		{"quote after a division", "a / b + 'q'", []string{"'q'"}},
	}
	runRegexCases(t, testCases)
}

func TestLiterals_DivisionBetweenTwoSlashes(t *testing.T) {
	t.Parallel()

	two := []string{`"a/"`, `"R$"`}
	testCases := []literalCase{
		{"postfix increment", `y = i++ / 2 + "a/" + x + "R$"`, two},
		{"postfix decrement", `y = a[0]-- / 2 + "a/" + x + "R$"`, two},
		{"non-null after a call", `y = f()! / 2 + "a/" + x + "R$"`, two},
		{"non-null after an index", `y = a[0]! / 2 + "a/" + x + "R$"`, two},
		{"non-null after a paren", `y = (a)! / 2 + "a/" + x + "R$"`, two},
		{"non-null after a word", `y = done! / 2 + "a/" + x + "R$"`, two},
		{"keyword property", `y = node.return / 2 + "a/" + x + "R$"`, two},
		{"optional keyword property", `y = a?.typeof / 2 + "a/" + x + "R$"`, two},
		{"closing tag", `<i>t</i> 'a/' b`, []string{`'a/'`}},
	}
	runRegexCases(t, testCases)
}

func runRegexCases(t *testing.T, testCases []literalCase) {
	t.Helper()
	comments, strs := regexSyntax()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := spanTexts(testCase.src, Literals(testCase.src, comments, strs).Spans)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Literals(%q) = %q, want %q", testCase.src, got, testCase.want)
			}
		})
	}
}

func TestLiterals_MarksRegexSpans(t *testing.T) {
	t.Parallel()

	comments, strs := regexSyntax()
	src := `x = /a/ + "b" + (/c/)`
	spans := Literals(src, comments, strs).Spans
	got := make([]bool, 0, len(spans))
	for _, span := range spans {
		got = append(got, span.Regex)
	}
	if want := []bool{true, false, true}; !slices.Equal(got, want) {
		t.Errorf("regex flags = %v, want %v", got, want)
	}
}

func TestLiterals_RegexHostileStaysLinear(t *testing.T) {
	t.Parallel()

	comments, strs := regexSyntax()
	const size = 1 << 20
	testCases := []struct {
		name string
		unit string
	}{
		{"assignment slashes", "=/"},
		{"division slashes", "a /"},
		{"class openers", "=/["},
		{"blank runs", "    /"},
		{"keyword runs", "return /"},
		{"escaped slashes", "=/\\/"},
		{"postfix operators", "i++ /"},
		{"non-null assertions", "a! /"},
		{"property keywords", "a.return /"},
		{"newline slashes", "a\n/"},
		{"paren runs", "(((("},
		{"if headers", "if (a) /"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			src := strings.Repeat(testCase.unit, size/len(testCase.unit))
			scan := &literalScan{src: src, comments: comments, syntax: strs}
			scan.run()
			if limit := 4 * len(src); scan.work > limit {
				t.Errorf("work = %d for %d bytes, limit %d", scan.work, len(src), limit)
			}
		})
	}
}
