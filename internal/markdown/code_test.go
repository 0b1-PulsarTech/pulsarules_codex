package markdown

import (
	"strings"
	"testing"
)

type regionCase struct {
	name string
	src  string
}

func TestCodeRegions(t *testing.T) {
	t.Parallel()

	testCases := []regionCase{
		{"prose is never code", "a P b\n"},
		{"fenced block hides its content", "text\n```\nS\n```\nP\n"},
		{"tilde fence works the same", "~~~\nS\n~~~\nP\n"},
		{"an unclosed fence runs to the end", "```\nS\n"},
		{"inline span hides its content", "see `S` P\n"},
		{"an unclosed backtick is not a span", "see ` P\n"},
		{"unmatched runs of distinct lengths protect nothing", "` `` ``` ```` P\n"},
		{"a two tick line is not a fence", "``\nP\n``\nP\n"},
		{"a tilde line does not close a backtick fence", "```\n~~~\nS\n```\nP\n"},
		{"a longer fence inside holds a shorter one", "````md\n```go\nS\n```\nS\n````\nP\n"},
		{"a closing fence longer than its opener closes", "```\nS\n`````\nP\n"},
		{"a fence in a list item", "- item\n  ```go\n  S\n  ```\nP\n"},
		{"spans of a run pair once", "`a``b`P``d\n"},
	}
	runRegionCases(t, testCases)
}

func TestCodeRegions_ReadingsDisagree(t *testing.T) {
	t.Parallel()

	testCases := []regionCase{
		{"an info line inside a fence", "```\nS\n```go\nM\n```\nM\n"},
		{"a fence inside a blockquote", "> ```go\n> M\n> ```\nP\n"},
		{"a fence inside a nested blockquote", "> > ```go\n> > M\n> > ```\nP\n"},
		{"a quoted closer inside a fence", "```markdown\n> ```go\n> S\n> ```\nM\n```\n"},
		{"a bare closer after a quoted fence", "> ```go\n> M\n```\nM\n"},
		{"a closer indented four spaces", "```text\nS\n    ```\nM\n```\n"},
		{"a tab-indented fence in a list", "- item\n\t```go\n\tM\n\t```\nP\n"},
		{"a list fence with a deep closer", "- a\n  ```go\n  S\n      ```\n  M\n  ```\n"},
		{"an indented code block after a blank line", "p\n\n    M\n\nP\n"},
		{"a tab-indented code line", "p\n\n\tM\n\nP\n"},
		{"indented code straight after a fence", "```\nS\n```\n    M\n"},
		{"a quote-only line counts as blank", "p\n>\n    M\n"},
		{"an indented line continuing a paragraph is prose", "p\n    P\n"},
		{"five spaces before a quote mark is no quote", "p\n     > ```\nP\n"},
		{"a nested list item", "- item\n\n    - M\n"},
		{"an escaped tick before a span", "see \\`M then `M` ok\n"},
		{"a backtick info string makes the run inline code", "```aS``` b\nM\n"},
	}
	runRegionCases(t, testCases)
}

func misread(code Code, offset int, char byte) string {
	certain, possible := code.Certain(offset), code.Possible(offset)
	switch {
	case certain && !possible:
		return "certain but not possible"
	case char == 'S' && !certain:
		return "not certain code, want certain"
	case char == 'M' && (certain || !possible):
		return "not a maybe, want possible but not certain"
	case char == 'P' && possible:
		return "reads as code, want prose"
	}
	return ""
}

func runRegionCases(t *testing.T, testCases []regionCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			code := CodeRegions(testCase.src)
			for offset, char := range []byte(testCase.src) {
				if problem := misread(code, offset, char); problem != "" {
					t.Errorf("offset %d: %s", offset, problem)
				}
			}
			if !strings.ContainsAny(testCase.src, "SMP") {
				t.Fatal("row has no marked byte")
			}
		})
	}
}
