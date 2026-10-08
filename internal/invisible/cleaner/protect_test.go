package cleaner

import (
	"os"
	"testing"
)

type markdownCase struct {
	name string
	body string
	want string
}

func untouched(name, body string) markdownCase {
	return markdownCase{name: name, body: body, want: body}
}

func TestCleanFile_LeavesMarkdownCodeAlone(t *testing.T) {
	t.Parallel()

	testCases := []markdownCase{
		untouched("backtick fence", "# Doc\n\n```go\nconst a = \"R$\u00A010\"\n```\n"),
		untouched("tilde fence", "~~~\nx\u200By\n~~~\n"),
		untouched("inline code", "see `R$\u00A010` here\n"),
		untouched("unclosed fence runs to the end", "```\nx\u200By\n"),
		{
			"prose around a fence is cleaned",
			"a\u00A0b\n\n```\nx\u200By\n```\n\nc\u200Bd\n",
			"a b\n\n```\nx\u200By\n```\n\ncd\n",
		},
		{"prose beside inline code is cleaned", "a\u200B `x\u200B` b\u200B\n", "a `x\u200B` b\n"},
	}
	runMarkdownCases(t, testCases)
}

func TestCleanFile_FollowsCommonMarkShapes(t *testing.T) {
	t.Parallel()

	testCases := []markdownCase{
		{
			"a longer outer fence holds an inner one",
			"````md\n```go\nx\u200By\n```\nz\u200B\n````\nc\u200Bd\n",
			"````md\n```go\nx\u200By\n```\nz\u200B\n````\ncd\n",
		},
		{
			"fence in a blockquote",
			"> ```go\n> x\u200By\n> ```\nc\u200Bd\n",
			"> ```go\n> x\u200By\n> ```\ncd\n",
		},
		{
			"fence in a list item",
			"- item\n  ```go\n  x\u200By\n  ```\nc\u200Bd\n",
			"- item\n  ```go\n  x\u200By\n  ```\ncd\n",
		},
		{
			"fence in a nested blockquote",
			"> > ```go\n> > x\u00A0y\n> > ```\n> > z\u00A0w\n",
			"> > ```go\n> > x\u00A0y\n> > ```\n> > z w\n",
		},
		{
			"indented code block",
			"p\n\n    x\u200By\n\nc\u200Bd\n",
			"p\n\n    x\u200By\n\ncd\n",
		},
	}
	runMarkdownCases(t, testCases)
}

func TestCleanFile_LeavesWhatMightBeCodeAlone(t *testing.T) {
	t.Parallel()

	testCases := []markdownCase{
		untouched(
			"a quoted closer inside a fence",
			"```markdown\n> ```go\n> x\n> ```\ncode\u00A0B\n```\n",
		),
		untouched("a bare closer after a quoted fence", "> ```go\n> x\n```\ncode\u00A0E\n"),
		untouched("an info line inside a fence", "```\nx\n```go\ny\n```\nc\u200Bd\n"),
		untouched("a closer indented four spaces", "```text\nline\n    ```\ncode\u00A0A\n```\n"),
		untouched("a tab-indented fence in a list", "- item\n\t```go\n\tcode\u00A0C\n\t```\n"),
		untouched("a nested list item", "- item\n\n    - nested\u00A0D prose\n"),
		untouched("an ordered list continuation", "1. Step\n\n    Prose\u00A0G continued\n"),
		untouched("an escaped tick before a span", "See \\`not code then `real\u00A0code` ok\n"),
		untouched(
			"a list fence with a deep closer",
			"- a\n  ```go\n  x\n      ```\n  y\u00A0H\n  ```\n",
		),
		untouched("a tab-indented code line", "p\n\n\tx\u00A0y\n"),
		untouched("a quote shifted three spaces", "   > ```\n> x\u00A0\n"),
		untouched("a tab after a space opens a fence", " \t```\nx\u00A0y\n"),
	}
	runMarkdownCases(t, testCases)
}

func runMarkdownCases(t *testing.T, testCases []markdownCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "doc.md", testCase.body)

			if _, err := New(root).CleanFile(path); err != nil {
				t.Fatalf("CleanFile: %v", err)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != testCase.want {
				t.Errorf("file = %q, want %q", body, testCase.want)
			}
		})
	}
}
