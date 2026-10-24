package tssyntax

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/sandbox"
)

type svelteScriptCase struct {
	name      string
	text      string
	wantFns   []string
	wantError string
}

func svelteScriptCases() []svelteScriptCase {
	return []svelteScriptCase{
		{
			name: "lang ts allows assertions",
			text: "<script lang=\"ts\">const a = <number>b;</script>",
		},
		{
			name:      "plain script reads assertions as jsx",
			text:      "<script>const a = <number>b;</script>",
			wantError: "line 1: ",
		},
		{
			name: "module then instance in order",
			text: "<script context=\"module\">function a() {}</script><script>function b() " +
				"{}</script>",
			wantFns: []string{"a", "b"},
		},
		{
			name:      "an error in the first script survives a clean second one",
			text:      "<script module>const = ;</script><script>function ok() {}</script>",
			wantFns:   []string{"ok"},
			wantError: "line 1: ",
		},
		{name: "no script at all", text: "<h1>hi</h1>"},
		{
			name:      "unterminated script",
			text:      "<script>let a;",
			wantError: "skipped: unterminated script block",
		},
		{
			name:    "extra scripts are ignored",
			text:    "<script>function a() {}</script><script>function b() {}</script>",
			wantFns: []string{"a"},
		},
	}
}

func svelteRecoveryCases() []svelteScriptCase {
	const nested = "a script block below the top level was not analysed"
	return []svelteScriptCase{
		{
			name: "a nested script is reported and the top level one still counts",
			text: "<div><script>function a() {}</script></div>" +
				"<script>function b() {}</script>",
			wantFns:   []string{"b"},
			wantError: nested,
		},
		{
			name:      "a script error wins over the nested script note",
			text:      "<script>const = ;</script><div><script>let a;</script></div>",
			wantError: "line 1: ",
		},
		{
			name: "a head script is not reported",
			text: "<svelte:head><script>let a;</script></svelte:head>" +
				"<script>function b() {}</script>",
			wantFns: []string{"b"},
		},
		{
			name: "a script after unclosed paragraphs",
			text: "<ul><li>a<li>b</ul>\n<p>unclosed\n" +
				"<script lang=\"ts\">function late(): void {}</script>",
			wantFns: []string{"late"},
		},
		{
			name:    "a script after glued less-than text",
			text:    "<p>a<b and more</p>\n<script>function late() {}</script>",
			wantFns: []string{"late"},
		},
		{
			name: "a script after a regex in a mustache",
			text: "<svelte:head>\n<script>function head() {}</script>\n</svelte:head>\n" +
				"<p>{x.replace(/'/g, \"\")}</p>\n<script>function late() {}</script>",
			wantFns: []string{"late"},
		},
	}
}

func svelteBraceCases() []svelteScriptCase {
	return []svelteScriptCase{
		{
			name:      "an unclosed brace after the script keeps its functions",
			text:      "<script>function early() {}</script>\n<p>{/*{/*",
			wantFns:   []string{"early"},
			wantError: "an unclosed brace stopped the markup scan",
		},
		{
			name:    "a component named Script does not hide the script",
			text:    "<Script src=\"x.js\">x</Script>\n<script>function late() {}</script>",
			wantFns: []string{"late"},
		},
	}
}

func TestSummarizeSvelteScripts(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(svelteScriptCases(), svelteRecoveryCases(), svelteBraceCases())
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: "a.svelte"}, testCase.text)
			if !strings.HasPrefix(got.ParseError, testCase.wantError) ||
				(testCase.wantError == "") != (got.ParseError == "") {
				t.Fatalf("parse error %q, want prefix %q", got.ParseError, testCase.wantError)
			}
			names := make([]string, 0, len(got.Functions))
			for _, fn := range got.Functions {
				names = append(names, fn.Name)
			}
			if strings.Join(names, ",") != strings.Join(testCase.wantFns, ",") {
				t.Fatalf("functions %v, want %v", names, testCase.wantFns)
			}
		})
	}
}

type groupCase struct {
	name       string
	text       string
	wantGroups []int
	wantNames  []string
}

func groupCases() []groupCase {
	return []groupCase{
		{
			name:       "one script keeps three groups",
			text:       "<script>import 'a'; import './b';</script>",
			wantGroups: []int{1, 2},
			wantNames:  []string{"node:", "external", "local"},
		},
		{
			name: "second script starts above the first",
			text: "<script module>import './helper';</script><script>import " +
				"'svelte';</script>",
			wantGroups: []int{2, 4},
			wantNames: []string{
				"node:", "external", "local",
				"node: (script 2)", "external (script 2)", "local (script 2)",
			},
		},
		{
			name:       "second script without imports adds no groups",
			text:       "<script module>import './helper';</script><script>let a;</script>",
			wantGroups: []int{2},
			wantNames:  []string{"node:", "external", "local"},
		},
		{
			name:       "first script without imports leaves the second plain",
			text:       "<script module>let a;</script><script>import 'svelte';</script>",
			wantGroups: []int{1},
			wantNames:  []string{"node:", "external", "local"},
		},
		{name: "no imports at all", text: "<script>let a;</script>"},
	}
}

func TestSummarizeSvelteImportGroups(t *testing.T) {
	t.Parallel()

	testCases := groupCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: "a.svelte"}, testCase.text)
			groups := make([]int, 0, len(got.Imports))
			for _, imported := range got.Imports {
				groups = append(groups, imported.Group)
			}
			if !slices.Equal(groups, testCase.wantGroups) {
				t.Fatalf("groups %v, want %v", groups, testCase.wantGroups)
			}
			if !slices.Equal(got.ImportGroups, testCase.wantNames) {
				t.Fatalf("group names %q, want %q", got.ImportGroups, testCase.wantNames)
			}
		})
	}
}

func TestSummarizeSvelteLocalAliases(t *testing.T) {
	t.Parallel()

	text := "<script module>import 'app/a';</script>" +
		"<script>import 'app/b'; import 'x';</script>"
	got := handleRequest(sandbox.Request{
		Source: syntax.Source{Path: "a.svelte", Content: []byte(text)},
		Local:  []string{"app/*"},
	})
	groups := make([]int, 0, len(got.Imports))
	for _, imported := range got.Imports {
		groups = append(groups, imported.Group)
	}
	if want := []int{2, 5, 4}; !slices.Equal(groups, want) {
		t.Fatalf("groups %v, want %v", groups, want)
	}
}

func TestSummarizeSvelteElses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want []syntax.Else
	}{
		{
			name: "template else blocks never count",
			text: "{#if ok}\n<p>a</p>\n{:else}\n<p>b</p>\n{/if}\n",
		},
		{
			name: "script else keeps the file line",
			text: "<p>x</p>\n{#if ok}\n{:else}\n{/if}\n<script>\nfunction f(a) {\nif (a) {\nreturn 1;\n} else {\nreturn 2;\n}\n}\n</script>\n",
			want: []syntax.Else{{Line: 9, AfterExit: true}},
		},
		{
			name: "two scripts merge their elses",
			text: "<script context=\"module\">\nif (a) {\n} else {\n}\n</script>\n<script>\nif (b) {\n} else {\n}\n</script>\n",
			want: []syntax.Else{{Line: 3}, {Line: 8}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeText(t, syntax.Source{Path: "a.svelte"}, testCase.text)
			if !slices.Equal(got.Elses, testCase.want) {
				t.Fatalf("elses %v, want %v", got.Elses, testCase.want)
			}
		})
	}
}
