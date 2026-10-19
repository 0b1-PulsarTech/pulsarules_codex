package lexer

import "testing"

func TestBlockEnd(t *testing.T) {
	t.Parallel()

	cBlock := BlockDelim{Open: "/*", Close: "*/"}
	docBlock := BlockDelim{Open: "/**", Close: "*/"}
	testCases := []struct {
		name       string
		lines      []string
		delim      BlockDelim
		wantEnd    int
		wantClosed bool
	}{
		{"closes on the opening line", []string{"/* a */", "b"}, cBlock, 0, true},
		{"closes on a later line", []string{"/*", " * a", " */", "b"}, cBlock, 2, true},
		{"closes on the last line", []string{"/*", " */"}, cBlock, 1, true},
		{"never closes", []string{"/*", " * a", "b"}, cBlock, 2, false},
		{"opener alone is not a closer", []string{"/*/", " */"}, cBlock, 1, true},
		{"doc opener alone is not a closer", []string{"/**/", " */"}, docBlock, 1, true},
		{"one line never closes", []string{"/*"}, cBlock, 0, false},
		{"indented opener", []string{"  /* a */"}, cBlock, 0, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			end, closed := BlockEnd(testCase.lines, testCase.delim)
			if end != testCase.wantEnd || closed != testCase.wantClosed {
				t.Fatalf(
					"BlockEnd = (%d, %v), want (%d, %v)",
					end, closed, testCase.wantEnd, testCase.wantClosed,
				)
			}
		})
	}
}

func TestCommentBody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		text string
		want string
	}{
		{"// note", "note"},
		{"//   spaced  ", "spaced"},
		{"/* block */", "block"},
		{"/* * star */", "star"},
		{"/** doc */", "doc"},
		{"/**/", ""},
		{"/**", ""},
		{" */", ""},
		{" * continued */", "continued"},
		{"  * continued", "continued"},
		{"# hash note", "hash note"},
		{"-- sql note", "sql note"},
		{"<!-- html note -->", "html note"},
		{"--[[ lua note ]]", "lua note"},
		{"//", ""},
		{"/// triple", "/ triple"},
		{"plain text", "plain text"},
		{"", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.text, func(t *testing.T) {
			t.Parallel()

			if got := CommentBody(testCase.text, AnyCommentSyntax()); got != testCase.want {
				t.Fatalf("CommentBody(%q) = %q, want %q", testCase.text, got, testCase.want)
			}
		})
	}
}

func TestCommentBody_OnlyTheGivenSyntax(t *testing.T) {
	t.Parallel()

	syntax := CommentSyntax{Line: []string{"#"}}
	if got := CommentBody("// note", syntax); got != "// note" {
		t.Fatalf("CommentBody = %q, want the text untouched", got)
	}
}

func TestTrimBOM(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want string
	}{
		{"with a BOM", "\xef\xbb\xbfpackage a", "package a"},
		{"without a BOM", "package a", "package a"},
		{"empty", "", ""},
		{"BOM only", "\xef\xbb\xbf", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := string(TrimBOM([]byte(testCase.src))); got != testCase.want {
				t.Fatalf("TrimBOM = %q, want %q", got, testCase.want)
			}
		})
	}
}
