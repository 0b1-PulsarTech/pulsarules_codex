package language

import (
	"slices"
	"testing"
)

func TestScanComments_Components(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(svelteCases(), endTagCases(), vueCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runComments(t, testCase)
		})
	}
}

func vueCases() []scanCase {
	return []scanCase{
		{
			"vue script after a stray brace",
			"vue",
			"<p>{</p>\n<script>\n// the old code\n</script>\n",
			1,
		},
		{"vue template comment", "vue", "<template><!-- the old code --></template>\n", 1},
		{"vue style block comment", "vue", "<style>\n/* the old code */\n</style>\n", 1},
		{"vue template slashes are text", "vue", "<p>{{ a // the old code }}</p>\n", 0},
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
		{
			"svelte script attributes on their own line",
			"svelte",
			"<script\n  lang=\"ts\"\n>\n// the old code\n</script>\n",
			1,
		},
		{"svelte script tag ends in CRLF", "svelte", "<script\r\n>\r\n// the old code\r\n" +
			"</script>\r\n", 1},
		{
			"svelte script tag ends in a tab",
			"svelte",
			"<script\t>\n// the old code\n</script>\n",
			1,
		},
		{
			"svelte self-closing script has no body",
			"svelte",
			"<script/>\n// the old code\n</script>\n",
			0,
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

func TestScanComments_Spans(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		ext    string
		source string
		want   []string
	}{
		{
			"dart block comments nest",
			".dart",
			"/* a /* b */ c */ x // d\n",
			[]string{"/* a /* b */ c */", "// d"},
		},
		{"go block comments do not nest", ".go", "/* a /* b */ c */\n", []string{"/* a /* b */"}},
		{"unknown extension", ".zzz", "// a\n", nil},
		{"no comment", ".py", "x = 1\n", nil},
		{"html comment", ".html", "<p>a</p><!-- b -->\n", []string{"<!-- b -->"}},
		{
			"svelte markup, script and style keep absolute offsets",
			".svelte",
			"<!-- a -->\n<script>\n// b\n</script>\n<style>\n/* c */\n</style>\n",
			[]string{"<!-- a -->", "// b", "/* c */"},
		},
		{
			"svelte script named in a comment opens no region",
			".svelte",
			"<!-- <script> -->\n<p>// not a comment</p>\n",
			[]string{"<!-- <script> -->"},
		},
		{
			"svelte style has no line comments",
			".svelte",
			"<style>\na { b: c } // d\n</style>\n",
			nil,
		},
		{
			"svelte html comment inside a script is text",
			".svelte",
			"<script>\nlet a = 1 <!-- b\n</script>\n",
			nil,
		},
		{
			"svelte unterminated comment runs to the end",
			".svelte",
			"<p>a</p><!-- b\nc",
			[]string{"<!-- b\nc"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, span := range ScanComments(testCase.ext, testCase.source) {
				got = append(got, testCase.source[span.Start:span.End])
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ScanComments(%q) = %q, want %q", testCase.source, got, testCase.want)
			}
		})
	}
}
