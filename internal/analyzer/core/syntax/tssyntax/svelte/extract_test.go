package svelte

import (
	"strings"
	"testing"
)

const (
	droppedErr  = "a script block below the top level was not analysed"
	unclosedErr = "an unclosed brace stopped the markup scan"
)

type wantScript struct {
	typeScript bool
	content    string
}

type extractCase struct {
	name    string
	text    string
	want    []wantScript
	wantErr string
}

func runExtract(t *testing.T, testCases []extractCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Extract(testCase.text)
			if (err == nil) != (testCase.wantErr == "") ||
				err != nil && err.Error() != testCase.wantErr {
				t.Fatalf("error %v, want %q", err, testCase.wantErr)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("%d scripts, want %d", len(got), len(testCase.want))
			}
			for idx, want := range testCase.want {
				if got[idx].TypeScript != want.typeScript {
					t.Fatalf("script %d typescript %v", idx, got[idx].TypeScript)
				}
				expected := blankedExcept(testCase.text, want.content)
				if got[idx].Text != expected {
					t.Fatalf("script %d text %q, want %q", idx, got[idx].Text, expected)
				}
			}
		})
	}
}

func blankedExcept(text, content string) string {
	out := []byte(strings.Repeat(" ", len(text)))
	for idx := range len(text) {
		if text[idx] == '\n' {
			out[idx] = '\n'
		}
	}
	copy(out[strings.Index(text, content):], content)
	return string(out)
}

func TestExtractScripts(t *testing.T) {
	t.Parallel()

	const moduleThenInstance = "<script context=\"module\" lang=\"ts\">\nMOD\n</script>\n" +
		"<script lang='ts'>\nINST\n</script>"
	runExtract(t, []extractCase{
		{name: "no script", text: "<h1>hi</h1>\n"},
		{
			name: "instance script keeps its bytes and the newlines",
			text: "<h1>x</h1>\n<script>\nlet INST;\n</script>\n",
			want: []wantScript{{content: "\nlet INST;\n"}},
		},
		{
			name: "module and instance",
			text: moduleThenInstance,
			want: []wantScript{
				{typeScript: true, content: "\nMOD\n"},
				{typeScript: true, content: "\nINST\n"},
			},
		},
		{
			name: "instance before module keeps document order",
			text: "<script>INST</script><script module>MOD</script>",
			want: []wantScript{{content: "INST"}, {content: "MOD"}},
		},
		{
			name: "uppercase tags and a greater-than inside an attribute",
			text: "<script data-x=\"a>b\" LANG=\"ts\">INST</SCRIPT>",
			want: []wantScript{{typeScript: true, content: "INST"}},
		},
		{
			name: "non ascii markup keeps byte offsets",
			text: "<p>\u0130\u0130</p><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "typescript type attribute",
			text: "<script type=\"text/typescript\">INST</script>",
			want: []wantScript{{typeScript: true, content: "INST"}},
		},
	})
}

func TestExtractIgnoredScripts(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name: "json script is data",
			text: "<script type=\"application/ld+json\">{\"a\": 1}</script><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "import map is data",
			text: "<script type=\"importmap\">{}</script><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "a comment holding a greater-than hides its script",
			text: "<!-- a > b <script>HID</script> --><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "style with an open brace is raw text",
			text: "<style>a {</style><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "self closing script has no content",
			text: "<script src=\"a.js\" />\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "a third script is ignored",
			text: "<script module>MOD</script><script>INST</script><script>EXTRA</script>",
			want: []wantScript{{content: "MOD"}, {content: "INST"}},
		},
		{
			name: "a second instance script is ignored",
			text: "<script>INST</script><script>EXTRA</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "a second module script is ignored",
			text: "<script context=\"module\">MOD</script><script module>EXTRA</script>",
			want: []wantScript{{content: "MOD"}},
		},
		{name: "script lookalike tag name", text: "<scripts>INST</scripts>"},
	})
}

func TestExtractMarkupShapes(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name:    "unclosed brace in an attribute stops the scan",
			text:    "<div a={x>y</div><script>INST</script>",
			wantErr: unclosedErr,
		},
		{
			name: "apostrophe in an unquoted attribute value",
			text: "<a title=it's>x</a>\n<script>INST</script>\n<p title='q'>y</p>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "script text in an attribute value",
			text: "<div title=\"<script>\">x</div><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "script text in a mustache string",
			text: "<p>{'<script>'}</p><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "greater-than in an attribute expression",
			text: "<div on:click={() => a > b}>x</div><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
	})
}

func TestExtractCaseAndText(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name: "component named Script first",
			text: "<Script src=\"x.js\">fallback</Script>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "nested component named Script",
			text: "<div><Script>fallback</Script></div>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "self-closing component named Script",
			text: "<Script />\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{name: "uppercase script alone is markup", text: "<SCRIPT>x</SCRIPT>"},
		{
			name: "less-than before a digit and before an equals sign",
			text: "<p>a <1 and b <= c</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "less-than digit then a greater-than later",
			text: "x <1 and y > z\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "less-than-or-equal then a greater-than later",
			text: "x <= y and z > w\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "keyword property then a division",
			text: "<div title={a.of / 2}>x</div> {b}\n<script>INST</script>\n<p>}</p>",
			want: []wantScript{{content: "INST"}},
		},
	})
}

func TestExtractNesting(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name:    "script inside an element",
			text:    "<div><script>EXTRA</script></div><script>INST</script>",
			want:    []wantScript{{content: "INST"}},
			wantErr: droppedErr,
		},
		{
			name: "script inside svelte head",
			text: "<svelte:head><script>EXTRA</script></svelte:head><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name:    "script inside a block",
			text:    "{#if a}<script>EXTRA</script>{/if}<script>INST</script>",
			want:    []wantScript{{content: "INST"}},
			wantErr: droppedErr,
		},
		{
			name: "script inside an element inside svelte head",
			text: "<svelte:head><div><script>EXTRA</script></div></svelte:head>" +
				"<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "script after an unrelated element",
			text: "<div>a</div><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "script after a closed block",
			text: "{#if a}<b>x</b>{:else}y{/if}<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
	})
}

func TestExtractLexemes(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name: "void elements do not nest",
			text: "<br><input type=text><img src=a><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "stray closers are ignored",
			text: "</div>{/if}<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "closer pops to the nearest match",
			text: "<div><span><div></div><script>EXTRA</script></span></div>" +
				"<script>INST</script>",
			want:    []wantScript{{content: "INST"}},
			wantErr: droppedErr,
		},
		{
			name: "commented out script is skipped",
			text: "<!-- <script>EXTRA</script> -->\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "script text in a style block is skipped",
			text: "<style>/* <script>EXTRA</script> */</style><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "doctype and comparison text",
			text: "<!doctype html><p>a < b</p><script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
	})
}

func TestExtractTruncated(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name:    "unterminated script",
			text:    "<script>let a;",
			wantErr: "unterminated script block",
		},
		{name: "unterminated comment hides the rest", text: "<!-- <script>x</script>"},
		{name: "unterminated opening tag", text: "<script lang=\"ts\""},
		{
			name:    "unterminated expression stops the scan",
			text:    "<script>x</script>{ <script>y</script>",
			want:    []wantScript{{content: "x"}},
			wantErr: unclosedErr,
		},
		{name: "unterminated data script", text: "<script type=\"application/json\">{"},
		{
			name:    "unterminated script inside an element",
			text:    "<div><script>let a;",
			wantErr: droppedErr,
		},
		{name: "unterminated style", text: "<style>a"},
	})
}

func TestExtractRecovery(t *testing.T) {
	t.Parallel()

	endTags := []string{
		"p", "li", "dt", "dd", "option", "optgroup", "tr", "td", "th", "thead", "tbody",
		"tfoot", "colgroup", "rb", "rt", "rtc", "rp",
	}
	testCases := make([]extractCase, 0, len(endTags)+5)
	for _, name := range endTags {
		testCases = append(testCases, extractCase{
			name: "unclosed " + name,
			text: "<" + name + ">x\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		})
	}
	testCases = append(testCases,
		extractCase{
			name: "list items without end tags",
			text: "<ul><li>a<li>b</ul>\n<p>unclosed\n<script lang=\"ts\">INST</script>",
			want: []wantScript{{typeScript: true, content: "INST"}},
		},
		extractCase{
			name: "less-than glued to text",
			text: "<p>a<b and more</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		extractCase{
			name: "regex with a quote in a mustache",
			text: "<svelte:head>\n<script>HEAD</script>\n</svelte:head>\n" +
				"<p>{x.replace(/'/g, \"\")}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		extractCase{
			name: "regex with a brace in a block",
			text: "{#if /}/.test(a)}x{/if}\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		extractCase{
			name:    "bold glued to text is still an element",
			text:    "<div>a<b>x</b><script>EXTRA</script></div><script>INST</script>",
			want:    []wantScript{{content: "INST"}},
			wantErr: droppedErr,
		},
	)
	runExtract(t, testCases)
}

func TestExtractMustacheFaults(t *testing.T) {
	t.Parallel()

	const hello = "\n  function hello() { return 1; }\n"
	runExtract(t, []extractCase{
		{
			name: "post-increment then a slash",
			text: "<p>{count++ / 2}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "decrement then a slash",
			text: "<p>{x-- / 2}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "comment-led mustache",
			text: "<p>{/* note */ a}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "line comment in a mustache",
			text: "<p>{// note\na}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "post-increment with a stray brace",
			text: "<p>{count++ / 2} }</p>\n<script>" + hello + "</script>\n",
			want: []wantScript{{content: hello}},
		},
		{
			name: "slash in an attribute expression",
			text: "<div title={a.of / 2}>x</div> {b}\n<script>" + hello + "</script>\n<p>}</p>\n",
			want: []wantScript{{content: hello}},
		},
		{
			name: "misread regex in an attribute",
			text: "<div title={n ? /b}>x</div>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "misread regex then a script on the next line",
			text: "<p>{a ? /b}</p>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
	})
}

func TestExtractMisreadMustache(t *testing.T) {
	t.Parallel()

	runExtract(t, []extractCase{
		{
			name: "post-increment in an attribute",
			text: "<div title={n++ / 2}>x</div>\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name: "block close after a misread mustache",
			text: "{#if a}{n ? /b}\n{/if}\n<script>INST</script>",
			want: []wantScript{{content: "INST"}},
		},
		{
			name:    "a truly unclosed brace still stops",
			text:    "<p>{a\n<script>INST</script>",
			wantErr: unclosedErr,
		},
		{
			name:    "a misread on the last line stops",
			text:    "<p>{n ? /b}</p><script>INST</script>",
			wantErr: unclosedErr,
		},
	})
}
