package lexer

import (
	"slices"
	"testing"
)

func TestLiterals_TaggedDelimiter(t *testing.T) {
	t.Parallel()

	comments := CommentSyntax{Line: []string{"//"}}
	strs := StringSyntax{Delims: []StringDelim{
		{Open: "`", Close: "`", Multiline: true},
		{Open: "`", Close: "`", Multiline: true, Raw: true, Tag: "raw"},
	}}
	testCases := []struct {
		name string
		src  string
		want []bool
	}{
		{"untagged", "a `x`", []bool{false}},
		{"tagged", "a raw`x`", []bool{true}},
		{"tagged after a blank", "a raw `x`", []bool{true}},
		{"tagged at the start", "raw`x`", []bool{true}},
		{"tagged after a punctuation byte", "(raw`x`)", []bool{true}},
		{"tag inside a longer word", "a myraw`x`", []bool{false}},
		{"tag as a prefix of a longer word", "a rawx`x`", []bool{false}},
		{"tag glued to a dollar", "$raw`x`", []bool{false}},
		{"tag glued to a non-ASCII letter", "\u00e9raw`x`", []bool{false}},
		{"tag spelled in a comment", "raw // raw\n`x`", []bool{true}},
		{"tagged then untagged", "raw`x` `y`", []bool{true, false}},
	}
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

func TestLiterals_LineReaderDelimiters(t *testing.T) {
	t.Parallel()

	noIdent := func(src string, at int) bool { return IdentBefore(src, at) == "" }
	testCases := []struct {
		name   string
		delims []StringDelim
		src    string
		want   []string
	}{
		{
			"hashes repeat in the closer",
			[]StringDelim{{Open: `r"`, Close: `"`, Raw: true, Hashes: true}},
			`r#"a"b"# r"c" r##"d"#"##`,
			[]string{`r#"a"b"#`, `r"c"`, `r##"d"#"##`},
		},
		{
			"hashes need the quote after the run",
			[]StringDelim{{Open: `r"`, Close: `"`, Raw: true, Hashes: true}},
			`r#x" r#`,
			nil,
		},
		{
			"doubled closer is one escaped closer",
			[]StringDelim{{Open: `@"`, Close: `"`, Doubled: true}},
			`@"a""b" c`,
			[]string{`@"a""b"`},
		},
		{
			"whole opens only when it closes on its line",
			[]StringDelim{{Open: `'`, Close: `'`, Escapes: true, Whole: true}},
			"a 'b\nc 'd'\ne 'f\\'\ng 'h\\\ni'",
			[]string{`'d'`},
		},
		{
			"char takes one character or one escape",
			[]StringDelim{{Open: `'`, Close: `'`, Escapes: true, Char: true}},
			"'a'\n'ab\n'\\n'\n'\\''\n'\\u{41}'\n'é'\n''\n<'a>(&'a x)",
			[]string{`'a'`, `'\n'`, `'\''`, `'\u{41}'`, `'é'`},
		},
		{
			"char ends where its own rule ends, not at an escaped quote",
			[]StringDelim{{Open: `'`, Close: `'`, Escapes: true, Char: true}},
			`'\"\'/*b`,
			[]string{`'\"\'`},
		},
		{
			"opens hook sees the source before the opener",
			[]StringDelim{{Open: `'`, Close: `'`, Opens: noIdent}},
			`a'b' 'c' 1'000'`,
			[]string{`'c'`},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			strs := StringSyntax{Delims: testCase.delims, EndAtNewline: true}
			got := spanTexts(testCase.src, Literals(testCase.src, CommentSyntax{}, strs).Spans)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Literals(%q) = %q, want %q", testCase.src, got, testCase.want)
			}
		})
	}
}

func TestLiterals_LineOpensGuardsLineComments(t *testing.T) {
	t.Parallel()

	afterBlank := func(src string, at int) bool { return at == 0 || src[at-1] == ' ' }
	comments := CommentSyntax{Line: []string{"#"}, LineOpens: afterBlank}
	testCases := []struct {
		name string
		src  string
		want []string
	}{
		{"after a blank", "a # b", []string{"# b"}},
		{"glued to a word", "a#b", nil},
		{"at the start", "# b", []string{"# b"}},
		{"only the guarded one is skipped", "a#b # c", []string{"# c"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := spanTexts(
				testCase.src,
				Literals(testCase.src, comments, StringSyntax{}).Comments,
			)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Comments of %q = %q, want %q", testCase.src, got, testCase.want)
			}
		})
	}
}

func TestIdentBefore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		at   int
		want string
	}{
		{"word before", "ab'x'", 2, "ab"},
		{"nothing before", "'x'", 0, ""},
		{"punctuation before", "a+'x'", 2, ""},
		{"digits and underscore count", "a_1'x'", 3, "a_1"},
		{"dollar is not an identifier byte", "$'x'", 1, ""},
		{"non-ASCII byte is not a prefix byte", "\u00e9'x'", 2, ""},
		{"stops at a blank", "a b'x'", 3, "b"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := IdentBefore(testCase.src, testCase.at); got != testCase.want {
				t.Errorf(
					"IdentBefore(%q, %d) = %q, want %q",
					testCase.src,
					testCase.at,
					got,
					testCase.want,
				)
			}
		})
	}
}
