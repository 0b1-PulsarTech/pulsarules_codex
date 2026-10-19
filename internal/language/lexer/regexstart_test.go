package lexer

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

func TestJSStrings_RegexAfterWords(t *testing.T) {
	t.Parallel()

	wantWords := strings.Fields(
		"return typeof case in of void yield await delete throw new " +
			"instanceof else do extends default",
	)
	wantParen := strings.Fields("if while for with")
	got := JSStrings().RegexAfter
	if !slices.Equal(got.Words, wantWords) {
		t.Errorf("string scanner words %v, want %v", got.Words, wantWords)
	}
	if !slices.Equal(got.ParenWords, wantParen) {
		t.Errorf("string scanner paren words %v, want %v", got.ParenWords, wantParen)
	}
}

func TestStartsExpression_IsExactlyTheScannerWords(t *testing.T) {
	t.Parallel()

	scanner := JSStrings().RegexAfter
	notStarters := strings.Fields(
		"export import let const var function class switch catch try finally this super " +
			"null true false async if while for with break continue returns",
	)
	for _, word := range scanner.Words {
		if !StartsExpression(word) {
			t.Errorf("StartsExpression(%q) = false, but JSStrings lists it", word)
		}
	}
	for _, word := range notStarters {
		if StartsExpression(word) {
			t.Errorf("StartsExpression(%q) = true, want false", word)
		}
	}
	for _, word := range scanner.ParenWords {
		if !IsHeaderWord(word) {
			t.Errorf("IsHeaderWord(%q) = false, but JSStrings lists it", word)
		}
	}
	if IsHeaderWord("switch") || IsHeaderWord("") {
		t.Error("IsHeaderWord accepts a word outside if/while/for/with")
	}
}

func TestStartsExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		word string
		want bool
	}{
		{"return", true},
		{"instanceof", true},
		{"default", true},
		{"class", false},
		{"returns", false},
		{"", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.word, func(t *testing.T) {
			t.Parallel()

			if got := StartsExpression(testCase.word); got != testCase.want {
				t.Fatalf("StartsExpression(%q) = %v, want %v", testCase.word, got, testCase.want)
			}
		})
	}
}

func TestIsHeaderWord(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		word string
		want bool
	}{
		{"if", true},
		{"while", true},
		{"for", true},
		{"with", true},
		{"switch", false},
		{"catch", false},
		{"iff", false},
		{"", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.word, func(t *testing.T) {
			t.Parallel()

			if got := IsHeaderWord(testCase.word); got != testCase.want {
				t.Fatalf("IsHeaderWord(%q) = %v, want %v", testCase.word, got, testCase.want)
			}
			inScanner := slices.Contains(JSStrings().RegexAfter.ParenWords, testCase.word)
			if inScanner != testCase.want {
				t.Fatalf(
					"scanner ParenWords has %q = %v, want %v",
					testCase.word,
					inScanner,
					testCase.want,
				)
			}
		})
	}
}

func TestIsIdentByte(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		char byte
		want bool
	}{
		{"lower", 'q', true},
		{"upper", 'Q', true},
		{"digit", '7', true},
		{"underscore", '_', true},
		{"dollar", '$', true},
		{"utf8 lead", 0xC3, true},
		{"dot", '.', false},
		{"space", ' ', false},
		{"quote", '\'', false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsIdentByte(testCase.char); got != testCase.want {
				t.Fatalf("IsIdentByte(%q) = %v, want %v", testCase.char, got, testCase.want)
			}
		})
	}
}

func TestLiterals_RegexOpensAfterEveryJSWord(t *testing.T) {
	t.Parallel()

	words := JSStrings().RegexAfter.Words
	testCases := make([]literalCase, 0, len(words))
	for _, word := range words {
		testCases = append(testCases, literalCase{
			"after " + word, "a " + word + ` /"/.test(s) + 'q'`, []string{`/"/`, `'q'`},
		})
	}
	comments, _ := testSyntax()
	scan := JSStrings()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := spanTexts(testCase.src, Literals(testCase.src, comments, scan).Spans)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("spans %q, want %q for %q", got, testCase.want, testCase.src)
			}
		})
	}
}
