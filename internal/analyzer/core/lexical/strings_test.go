package lexical

import (
	"slices"
	"strings"
	"testing"
)

func spanTexts(src string, spans []Span) []string {
	texts := make([]string, 0, len(spans))
	for _, span := range spans {
		texts = append(texts, src[span.Start:span.End])
	}
	return texts
}

func testSyntax() (CommentSyntax, StringSyntax) {
	comments := CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	}
	strs := StringSyntax{Delims: []StringDelim{
		{Open: `"`, Close: `"`, Escapes: true, Continues: true},
		{Open: `'`, Close: `'`, Escapes: true, Continues: true, Interp: "${"},
		{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: "${"},
		{Open: `'''`, Close: `'''`, Multiline: true},
		{Open: `r'`, Close: `'`, Raw: true},
	}}
	return comments, strs
}

type literalCase struct {
	name string
	src  string
	want []string
}

func runLiteralCases(t *testing.T, testCases []literalCase) {
	t.Helper()
	comments, strs := testSyntax()
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

func TestLiterals(t *testing.T) {
	t.Parallel()

	testCases := []literalCase{
		{"empty file", "", nil},
		{"no literal", "x := 1 + 2\n", nil},
		{"double quoted", `a := "hi" + b`, []string{`"hi"`}},
		{"escaped quote stays inside", `a := "h\"i" + "j"`, []string{`"h\"i"`, `"j"`}},
		{"escaped backslash closes", `a := "h\\" + "j"`, []string{`"h\\"`, `"j"`}},
		{"raw string has no escapes", `a := r'x\' + 'y'`, []string{`r'x\'`, `'y'`}},
		{"triple quote spans lines", "a := '''x\n'y'\n''' + 1", []string{"'''x\n'y'\n'''"}},
		{"template spans lines", "a := `x\ny`", []string{"`x\ny`"}},
		{"line comment hides a quote", "// \"no\nb := \"yes\"", []string{`"yes"`}},
		{"block comment hides a quote", "/* ' */ b := 'ok'", []string{`'ok'`}},
		{"quote after a comment on the same line", "a /* x */ := 'z'", []string{`'z'`}},
		{"comment marker inside a literal", `a := "// not"`, []string{`"// not"`}},
		{"unterminated block comment", "/* \"x\"", nil},
		{"unterminated single line ends at newline", "a := 'x\nb := 'y'", []string{"'x", "'y'"}},
		{"unterminated triple runs to EOF", "'''abc\n", []string{"'''abc\n"}},
		{"unterminated template runs to EOF", "`abc", []string{"`abc"}},
		{"trailing backslash at EOF", `"ab\`, []string{`"ab\`}},
		{"LF continuation stays inside", "'a\\\nb' + 'c'", []string{"'a\\\nb'", "'c'"}},
		{"CRLF continuation stays inside", "'a\\\r\nb' + 'c'", []string{"'a\\\r\nb'", "'c'"}},
		{"backslash before CR alone", "'a\\\rb' + 'c'", []string{"'a\\\rb'", "'c'"}},
		{"CRLF ends an unterminated literal at the newline", "'x\r\n'y'", []string{"'x\r", "'y'"}},
		{
			"multibyte before the literal",
			"a := \"\u00E9\u00E9\" + 'z'",
			[]string{"\"\u00E9\u00E9\"", "'z'"},
		},
	}
	runLiteralCases(t, testCases)
}

func TestLiterals_Interpolation(t *testing.T) {
	t.Parallel()

	testCases := []literalCase{
		{
			"interpolation is code",
			"'a ${b + 'c'} d'",
			[]string{"'a ", "'c'", " d'"},
		},
		{
			"nested template inside interpolation",
			"`a ${ `in ${ x } ner` } z`",
			[]string{"`a ", "`in ", " ner`", " z`"},
		},
		{
			"braces inside interpolation code",
			"'a ${ {k: 1}['k'] } b'",
			[]string{"'a ", "'k'", " b'"},
		},
		{"comment inside interpolation code", "'a ${ // '\n x } b'", []string{"'a ", " b'"}},
		{"unterminated interpolation", "'a ${ b", []string{"'a "}},
		{"stray closing brace is code", "} 'x'", []string{"'x'"}},
	}
	runLiteralCases(t, testCases)
}

func TestLiterals_Complete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want bool
	}{
		{"empty file", "", true},
		{"closed literals", `a := "x" + 'y' + ` + "`z`", true},
		{"closed block comment", "/* a */ 'x'", true},
		{"unclosed block comment", "/* a 'x'", false},
		{"unclosed template", "a := `half\nwritten", false},
		{"unclosed triple quote", "'''abc\n", false},
		{"unclosed single line at EOF", "a := 'abc", false},
		{"single line cut by newline", "a := 'abc\nb := 1\n", false},
		{"single line cut by CRLF", "a := \"abc\r\nb := 1\n", false},
		{"trailing backslash at EOF", `"ab\`, false},
		{"open interpolation frame", "'a ${ b", false},
		{"closed interpolation frame", "'a ${ b } c'", true},
		{"line comment to EOF is whole", "x // 'abc", true},
	}
	comments, strs := testSyntax()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := Literals(testCase.src, comments, strs).Complete; got != testCase.want {
				t.Errorf("Complete = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}

func TestLiterals_MarksRawSpans(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want []bool
	}{
		{"interpreted is not raw", `a := "x"`, []bool{false}},
		{"raw delimiter", `a := r'x\'`, []bool{true}},
		{"mixed in order", `"a" r'b' 'c' r'd'`, []bool{false, true, false, true}},
		{"nested template stays not raw", "`a ${ r'b' } c`", []bool{false, true, false}},
	}
	comments, strs := testSyntax()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got []bool
			for _, span := range Literals(testCase.src, comments, strs).Spans {
				got = append(got, span.Raw)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("raw flags = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}

func TestLiterals_HostileInputStaysLinear(t *testing.T) {
	t.Parallel()

	comments, strs := testSyntax()
	const size = 1 << 20
	testCases := []struct {
		name string
		unit string
	}{
		{"double quotes", `"`},
		{"triple quotes", `'''`},
		{"interpolation openers", "`${"},
		{"bare openers", "${"},
		{"backticks", "`"},
		{"comment openers", "/* "},
		{"line comment openers", "//"},
		{"escape pairs", "\"\\"},
		{"escaped CRLF", "'\\\r\n"},
		{"unterminated lines", "'x\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			src := strings.Repeat(testCase.unit, size/len(testCase.unit))
			scan := &literalScan{src: src, comments: comments, syntax: strs}
			scan.run()
			if limit := 3 * len(src); scan.work > limit {
				t.Errorf("work = %d for %d bytes, limit %d", scan.work, len(src), limit)
			}
		})
	}
}

func TestLiterals_ContinuationIsPerLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		src       string
		continues bool
		wantFull  bool
	}{
		{"js LF continuation", "'a\\\nb'", true, true},
		{"js CRLF continuation", "'a\\\r\nb'", true, true},
		{"strict LF backslash leaves it open", "'a\\\nb'", false, false},
		{"strict CRLF backslash leaves it open", "'a\\\r\nb'", false, false},
		{"strict escaped quote still escapes", "'a\\'b'", false, true},
	}
	comments := CommentSyntax{Line: []string{"//"}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			strs := StringSyntax{Delims: []StringDelim{
				{Open: "'", Close: "'", Escapes: true, Continues: testCase.continues},
			}}
			got := Literals(testCase.src, comments, strs).Complete
			if got != testCase.wantFull {
				t.Errorf("Complete = %v, want %v for %q", got, testCase.wantFull, testCase.src)
			}
		})
	}
}

func TestLiterals_CutVouchesOnlyForALineCut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want int
		done bool
		eof  bool
	}{
		{"a closed file has nothing to vouch for", "a = 'x'\n", 0, true, false},
		{"a quote cut at offset zero", "'x\ny'", 0, false, false},
		{"a quote cut at the newline", "a = 1\nb = 'x\nc = 'y'\n", 10, false, false},
		{"a quote cut at the newline of a CRLF", "a = 'x\r\nb\r\n", 4, false, false},
		{"a literal that runs to EOF", "a = 'x", 0, false, true},
		{"a template that runs to EOF", "a = `x", 0, false, true},
		{"a comment that runs to EOF", "a = 1 /* x", 0, false, true},
		{"an interpolation that never closes", "a = `x ${b", 0, false, true},
		{"the first cut stands when EOF follows", "a = 'x\nb = `y", 4, false, false},
		{"the first cut stands when a line cut follows", "a = 'x\nb = 'y\n", 4, false, false},
	}
	comments := CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	}
	strs := StringSyntax{Delims: []StringDelim{
		{Open: "'", Close: "'", Escapes: true},
		{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: "${"},
	}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Literals(testCase.src, comments, strs)
			if got.Complete != testCase.done || got.Cut != testCase.want ||
				got.ToEOF != testCase.eof {
				t.Errorf(
					"Complete, Cut, ToEOF = %v, %d, %v, want %v, %d, %v for %q",
					got.Complete, got.Cut, got.ToEOF,
					testCase.done, testCase.want, testCase.eof, testCase.src,
				)
			}
		})
	}
}
