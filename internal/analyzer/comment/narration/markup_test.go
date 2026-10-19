package narration

import (
	"slices"
	"testing"
)

func TestMarkupRegions(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(svelteCases(), endTagCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func endTagCases() []scanCase {
	return []scanCase{
		{
			"svelte style end tag leaves the region",
			"svelte",
			"<style>\n</style>\n<p>/* the old code */</p>\n",
			0,
		},
		{
			"svelte script and comment on one line",
			"svelte",
			"<script>let a = 1 <!-- the old code\n</script>\n",
			0,
		},
		{"svelte script at end of line", "svelte", "<script\n// the old code\n</script>\n", 1},
		{"svelte script tag ends in CRLF", "svelte", "<script\r\n// the old code\r\n" +
			"</script>\r\n", 1},
		{
			"svelte script tag ends in a tab",
			"svelte",
			"<script\t>\n// the old code\n</script>\n",
			1,
		},
		{
			"svelte script tag ends in a slash",
			"svelte",
			"<script/>\n// the old code\n</script>\n",
			1,
		},
		{"svelte end tag ends in CRLF", "svelte", "<script>\n</script\r\n" +
			"<p>// the old code</p>\n", 0},
		{"svelte html comment glued to a word", "svelte", "<p>x</p><!--the old code-->\n", 1},
		{"svelte html comment at end of line", "svelte", "<!--\nthe old code\n-->\n", 1},
		{
			"py style tag does not end the file",
			"py",
			"HTML = \"\"\"\n<style>\n</style>\n\"\"\"\n# this used to be global\n",
			1,
		},
		{
			"sh end tag does not end the file",
			"sh",
			"cat <<EOF\n</script>\nEOF\n# we used to call curl\n",
			1,
		},
		{
			"php end tag does not end the file",
			"php",
			"<script>\n</script>\n<?php\n// the old code ran here\n",
			1,
		},
	}
}

func svelteCases() []scanCase {
	return []scanCase{
		{
			"svelte script string",
			"svelte",
			"<script>\nlet a = \"x // the old code\";\n</script>\n",
			0,
		},
		{"svelte script comment", "svelte", "<script>\n// the old code\n</script>\n", 1},
		{"svelte template slashes are text", "svelte", "<p>see // the old code</p>\n", 0},
		{"svelte template block opener is text", "svelte", "<p>a /* the old code */ b</p>\n", 0},
		{
			"svelte tag name prefix is not a script",
			"svelte",
			"<p>use <scriptable> and // the old code</p>\n",
			0,
		},
		{"svelte html comment", "svelte", "<p>x</p>\n<!-- the old code -->\n", 1},
		{"svelte apostrophe in markup", "svelte", "<p>don't</p>\n<!-- the old code -->\n", 1},
		{
			"svelte html comment in script is text",
			"svelte",
			"<script>\nlet a = 1 <!-- the old code\n</script>\n",
			0,
		},
		{"svelte style block comment", "svelte", "<style>\n/* the old code */\n</style>\n", 1},
		{
			"svelte style slashes are not comments",
			"svelte",
			"<style>\na { b: url(//x) } // the old code\n</style>\n",
			0,
		},
		{
			"svelte script end tag needs a boundary",
			"svelte",
			"<script>\nlet a = b </scripts; // the old code\n</script>\n",
			1,
		},
	}
}
