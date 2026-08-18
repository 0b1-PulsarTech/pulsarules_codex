package knowledge

import "testing"

// TestFirstSentence asserts the text is cut at (and including) the first
// sentence-ending period - one followed by whitespace or end of string - and
// that a period embedded in a dotted identifier or file extension is skipped
// rather than mistaken for a sentence boundary.
func TestFirstSentence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"single sentence", "One thing. And more.", "One thing."},
		{"no period", "no period here", "no period here"},
		{"empty", "", ""},
		{"leading period", ". rest", "."},
		{"trailing period with nothing after it", "Done.", "Done."},
		{
			"test file extension mid-sentence",
			"Colocated same-package tests, one _test.go per source file, table-driven.",
			"Colocated same-package tests, one _test.go per source file, table-driven.",
		},
		{
			"dotted identifier mid-sentence",
			"All outbound HTTP through one gateway; no http.DefaultClient; retries only.",
			"All outbound HTTP through one gateway; no http.DefaultClient; retries only.",
		},
		{
			"method call mid-sentence",
			"Tracing via an injected Tracer; spans with defer span.End(); errors recorded.",
			"Tracing via an injected Tracer; spans with defer span.End(); errors recorded.",
		},
		{
			"markdown file before a comma",
			"Challenge terms against CONTEXT.md, sharpen fuzzy language, then record it.",
			"Challenge terms against CONTEXT.md, sharpen fuzzy language, then record it.",
		},
		{
			"go file before a closing paren",
			"Fixed file set, unexported DTOs, mapper.go, DI binding, and config wiring.",
			"Fixed file set, unexported DTOs, mapper.go, DI binding, and config wiring.",
		},
		{
			"genuine sentence break",
			"First sentence here. Second sentence follows.",
			"First sentence here.",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := FirstSentence(testCase.input); got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestFirstBlockquote asserts the summary blockquote is extracted with its
// "> " markers stripped and consecutive lines joined, and that a body with no
// blockquote (or only an H1) yields an empty summary.
func TestFirstBlockquote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"no blockquote", "\n# Title\n\nSome prose with no quote.\n", ""},
		{"only an H1", "\n# Title\n", ""},
		{"single line", "\n# Title\n\n> One-line summary.\n\nRest of body.\n", "One-line summary."},
		{
			"multi line",
			"\n# Title\n\n> First line of the\n> summary continues here.\n\nRest of body.\n",
			"First line of the summary continues here.",
		},
		{
			"blockquote not at the top",
			"\n# Title\n\nSome lead-in prose.\n\n> Summary comes after prose.\n",
			"Summary comes after prose.",
		},
		{
			"blockquote only inside a section is not the summary",
			"\n# Title\n\nNo top-level summary here.\n\n" +
				"{{define \"forbidden\"}}\n> STRAY NOTE, NOT A SUMMARY\n- Do not do the thing.\n{{end}}\n",
			"",
		},
		{
			"top-level summary survives a later section blockquote",
			"\n# Title\n\n> The real summary.\n\n" +
				"{{define \"forbidden\"}}\n> Not the summary.\n{{end}}\n",
			"The real summary.",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := firstBlockquote(testCase.input); got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
