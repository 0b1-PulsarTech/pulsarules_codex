package knowledge

import "testing"

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
